package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type Updater interface {
	CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateRelease, error)
	DownloadAndApply(ctx context.Context, downloadURL string, targetPath string) error
	Restart(targetPath string) error
}

type UpdateRelease struct {
	Version     string
	DownloadURL string
	AssetName   string
	Size        int64
}

type gitHubRelease struct {
	TagName string        `json:"tag_name"`
	Assets  []gitHubAsset `json:"assets"`
}

type gitHubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

type GitHubUpdater struct {
	BaseURL     string
	HTTPClient  *http.Client
	GOOS        string
	GOARCH      string
	restartFunc func(targetPath string) error
}

func NewGitHubUpdater() *GitHubUpdater {
	return &GitHubUpdater{
		BaseURL: "https://api.github.com",
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		GOOS:   runtime.GOOS,
		GOARCH: runtime.GOARCH,
	}
}

func (u *GitHubUpdater) getBaseURL() string {
	if u.BaseURL != "" {
		return strings.TrimSuffix(u.BaseURL, "/")
	}
	return "https://api.github.com"
}

func (u *GitHubUpdater) getHTTPClient() *http.Client {
	if u.HTTPClient != nil {
		return u.HTTPClient
	}
	return &http.Client{
		Timeout: 10 * time.Second,
	}
}

func (u *GitHubUpdater) getGOOS() string {
	if u.GOOS != "" {
		return u.GOOS
	}
	return runtime.GOOS
}

func (u *GitHubUpdater) getGOARCH() string {
	if u.GOARCH != "" {
		return u.GOARCH
	}
	return runtime.GOARCH
}

func normalizeSemver(v string) string {
	v = strings.TrimSpace(v)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

func (u *GitHubUpdater) CheckForUpdate(ctx context.Context, currentVersion string) (*UpdateRelease, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	url := fmt.Sprintf("%s/repos/brotherlogic/gramophile/releases/latest", u.getBaseURL())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create update request: %w", err)
	}
	req.Header.Set("User-Agent", "gramophile-tui")

	resp, err := u.getHTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to check for updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api returned status %d", resp.StatusCode)
	}

	var release gitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release response: %w", err)
	}

	latestVersion := normalizeSemver(release.TagName)
	if !semver.IsValid(latestVersion) {
		return nil, fmt.Errorf("release tag %q is not valid semver", release.TagName)
	}

	normCurrentVersion := normalizeSemver(currentVersion)
	if currentVersion != "" && semver.IsValid(normCurrentVersion) {
		if semver.Compare(latestVersion, normCurrentVersion) <= 0 {
			return nil, nil
		}
	}

	expectedAssetName := fmt.Sprintf("gramophile_%s_%s", u.getGOOS(), u.getGOARCH())
	if u.getGOOS() == "windows" {
		expectedAssetName += ".exe"
	}

	for _, asset := range release.Assets {
		if asset.Name == expectedAssetName {
			return &UpdateRelease{
				Version:     latestVersion,
				DownloadURL: asset.BrowserDownloadURL,
				AssetName:   asset.Name,
				Size:        asset.Size,
			}, nil
		}
	}

	return nil, fmt.Errorf("no asset matching %s found for release %s", expectedAssetName, latestVersion)
}

func (u *GitHubUpdater) DownloadAndApply(ctx context.Context, downloadURL string, targetPath string) error {
	var err error
	if targetPath == "" {
		targetPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}
		targetPath, err = filepath.EvalSymlinks(targetPath)
		if err != nil {
			return fmt.Errorf("failed to resolve symlink for executable: %w", err)
		}
	}

	dir := filepath.Dir(targetPath)
	filename := filepath.Base(targetPath)

	// Validate write permissions on target directory
	probeFile := filepath.Join(dir, fmt.Sprintf(".perm_check_%d", os.Getpid()))
	f, err := os.OpenFile(probeFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("target directory is not writable: %w", err)
	}
	f.Close()
	_ = os.Remove(probeFile)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "gramophile-tui")

	resp, err := u.getHTTPClient().Do(req)
	if err != nil {
		return fmt.Errorf("failed to download release binary: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d while downloading update", resp.StatusCode)
	}

	tmpPath := filepath.Join(dir, fmt.Sprintf(".%s.tmp.%d", filename, os.Getpid()))
	tmpFile, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}

	success := false
	defer func() {
		if !success {
			_ = os.Remove(tmpPath)
		}
	}()

	n, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write update binary: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	if resp.ContentLength > 0 && n != resp.ContentLength {
		return fmt.Errorf("download size mismatch: got %d bytes, expected %d", n, resp.ContentLength)
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to set executable permissions: %w", err)
	}

	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("failed to atomically replace binary: %w", err)
	}

	success = true
	return nil
}

func (u *GitHubUpdater) Restart(targetPath string) error {
	if u.restartFunc != nil {
		return u.restartFunc(targetPath)
	}
	if targetPath == "" {
		var err error
		targetPath, err = os.Executable()
		if err != nil {
			return fmt.Errorf("failed to determine executable path: %w", err)
		}
	}
	return restartBinary(targetPath)
}
