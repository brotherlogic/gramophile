package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDocsFilesExist(t *testing.T) {
	// We are running in the workspace directory. We can check paths relative to the root.
	// But tests might run in different directories depending on go test invocation, so let's locate the workspace root.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	// Traverse up to find the root containing go.mod
	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("Failed to locate workspace root starting from %s", dir)
		}
		root = parent
	}

	// Define required files
	requiredFiles := []string{
		"mkdocs.yml",
		"nginx.conf",
		"Dockerfile.docs",
		filepath.Join("docs", "index.md"),
		filepath.Join("docs", "onboarding.md"),
		filepath.Join("docs", "syncing.md"),
		filepath.Join("docs", "organisations.md"),
	}

	for _, file := range requiredFiles {
		path := filepath.Join(root, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Required file %s does not exist", file)
		}
	}
}

func TestNginxConfig(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("Failed to locate workspace root starting from %s", dir)
		}
		root = parent
	}

	nginxPath := filepath.Join(root, "nginx.conf")
	data, err := os.ReadFile(nginxPath)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skip("nginx.conf not found, handled by TestDocsFilesExist")
		}
		t.Fatalf("Failed to read nginx.conf: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "listen 8080;") {
		t.Errorf("nginx.conf does not configure listening on port 8080")
	}
	if !strings.Contains(content, "error_page 404 /404.html;") {
		t.Errorf("nginx.conf does not configure custom 404 page fallback")
	}
}

func TestDocsFilesArePopulated(t *testing.T) {
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}

	root := dir
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatalf("Failed to locate workspace root starting from %s", dir)
		}
		root = parent
	}

	requiredFiles := []string{
		filepath.Join("docs", "index.md"),
		filepath.Join("docs", "onboarding.md"),
		filepath.Join("docs", "syncing.md"),
		filepath.Join("docs", "organisations.md"),
	}

	for _, file := range requiredFiles {
		path := filepath.Join(root, file)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read %s: %v", file, err)
		}
		content := string(data)
		lines := strings.Split(content, "\n")
		if len(lines) < 15 {
			t.Errorf("File %s seems to be a placeholder (only has %d lines, expected at least 15)", file, len(lines))
		}
		if strings.Contains(content, "placeholder") || strings.Contains(content, "This page will provide") {
			t.Errorf("File %s still contains placeholder/intro text", file)
		}
	}
}
