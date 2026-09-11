package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestUpdater_CheckForUpdate_NewerAvailable(t *testing.T) {
	expectedAsset := fmt.Sprintf("gramophile_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		expectedAsset += ".exe"
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "gramophile-tui" {
			t.Errorf("expected User-Agent gramophile-tui, got %s", r.Header.Get("User-Agent"))
		}
		if r.URL.Path != "/repos/brotherlogic/gramophile/releases/latest" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.2.0",
			"assets": []map[string]any{
				{
					"name":                 expectedAsset,
					"browser_download_url": "https://example.com/download/" + expectedAsset,
					"size":                 1024,
				},
			},
		})
	}))
	defer server.Close()

	updater := &GitHubUpdater{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
		GOOS:       runtime.GOOS,
		GOARCH:     runtime.GOARCH,
	}

	rel, err := updater.CheckForUpdate(context.Background(), "v1.1.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel == nil {
		t.Fatal("expected update release, got nil")
	}
	if rel.Version != "v1.2.0" {
		t.Errorf("expected version v1.2.0, got %s", rel.Version)
	}
	if rel.AssetName != expectedAsset {
		t.Errorf("expected asset name %s, got %s", expectedAsset, rel.AssetName)
	}
	if rel.DownloadURL != "https://example.com/download/"+expectedAsset {
		t.Errorf("expected download URL https://example.com/download/%s, got %s", expectedAsset, rel.DownloadURL)
	}
	if rel.Size != 1024 {
		t.Errorf("expected size 1024, got %d", rel.Size)
	}
}

func TestUpdater_CheckForUpdate_AlreadyLatest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.2.0",
			"assets":   []map[string]any{},
		})
	}))
	defer server.Close()

	updater := &GitHubUpdater{
		BaseURL:    server.URL,
		HTTPClient: server.Client(),
	}

	// Test with equal version
	rel, err := updater.CheckForUpdate(context.Background(), "v1.2.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != nil {
		t.Fatalf("expected nil when already at latest version, got %+v", rel)
	}

	// Test with newer current version (e.g. ahead of release)
	rel, err = updater.CheckForUpdate(context.Background(), "v1.3.0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rel != nil {
		t.Fatalf("expected nil when current version is ahead, got %+v", rel)
	}
}

func TestUpdater_CheckForUpdate_RateLimited(t *testing.T) {
	for _, statusCode := range []int{http.StatusForbidden, http.StatusTooManyRequests} {
		t.Run(fmt.Sprintf("status_%d", statusCode), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "rate limit exceeded", statusCode)
			}))
			defer server.Close()

			updater := &GitHubUpdater{
				BaseURL:    server.URL,
				HTTPClient: server.Client(),
			}

			rel, err := updater.CheckForUpdate(context.Background(), "v1.0.0")
			if err == nil {
				t.Fatalf("expected error on HTTP %d, got nil", statusCode)
			}
			if rel != nil {
				t.Fatalf("expected nil release on rate limit, got %+v", rel)
			}
		})
	}
}

func TestUpdater_DownloadAndApply_Success(t *testing.T) {
	payload := []byte("#!/bin/sh\necho updated\n")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		w.Write(payload)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "gramophile")
	if err := os.WriteFile(targetPath, []byte("old"), 0755); err != nil {
		t.Fatalf("failed to write initial file: %v", err)
	}

	updater := &GitHubUpdater{
		HTTPClient: server.Client(),
	}

	err := updater.DownloadAndApply(context.Background(), server.URL, targetPath)
	if err != nil {
		t.Fatalf("DownloadAndApply failed: %v", err)
	}

	content, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("failed to read target file: %v", err)
	}
	if string(content) != string(payload) {
		t.Errorf("expected content %q, got %q", string(payload), string(content))
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		t.Fatalf("failed to stat target file: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("expected permissions 0755, got %v", info.Mode().Perm())
	}
}

func TestUpdater_DownloadAndApply_ReadOnlyDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory permissions differ on Windows")
	}

	payload := []byte("new-binary")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.Write(payload)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	readOnlyDir := filepath.Join(tempDir, "readonly")
	if err := os.Mkdir(readOnlyDir, 0555); err != nil {
		t.Fatalf("failed to create read-only dir: %v", err)
	}
	defer os.Chmod(readOnlyDir, 0755)

	targetPath := filepath.Join(readOnlyDir, "gramophile")

	updater := &GitHubUpdater{
		HTTPClient: server.Client(),
	}

	err := updater.DownloadAndApply(context.Background(), server.URL, targetPath)
	if err == nil {
		t.Fatal("expected error when downloading into read-only directory, got nil")
	}
}

func TestUpdater_DownloadAndApply_SizeMismatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		w.Write([]byte("short"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	targetPath := filepath.Join(tempDir, "gramophile")

	updater := &GitHubUpdater{
		HTTPClient: server.Client(),
	}

	err := updater.DownloadAndApply(context.Background(), server.URL, targetPath)
	if err == nil {
		t.Fatal("expected error on size mismatch, got nil")
	}
}

func TestUpdater_Restart_CustomFunc(t *testing.T) {
	restarted := false
	updater := &GitHubUpdater{
		restartFunc: func(targetPath string) error {
			restarted = true
			return nil
		},
	}
	err := updater.Restart("/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !restarted {
		t.Fatal("expected restartFunc to be called")
	}
}
