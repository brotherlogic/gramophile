package main

import (
	"os"
	"runtime/debug"
	"testing"
)

func TestGetHost_Default(t *testing.T) {
	os.Unsetenv("GRAMOPHILE_HOST")
	expected := "gramophile-grpc.brotherlogic-backend.com:80"
	actual := getHost()
	if actual != expected {
		t.Errorf("Expected host %q, got %q", expected, actual)
	}
}

func TestGetHost_EnvOverride(t *testing.T) {
	customHost := "localhost:8080"
	os.Setenv("GRAMOPHILE_HOST", customHost)
	defer os.Unsetenv("GRAMOPHILE_HOST")

	actual := getHost()
	if actual != customHost {
		t.Errorf("Expected host %q, got %q", customHost, actual)
	}
}

func TestResolveVersion_BuildFlag(t *testing.T) {
	oldVersion := Version
	defer func() { Version = oldVersion }()

	Version = "v1.2.3"
	got := resolveVersion()
	if got != "v1.2.3" {
		t.Errorf("Expected stamped version %q, got %q", "v1.2.3", got)
	}

	Version = "v0.1.0"
	got = resolveVersion()
	if got != "v0.1.0" {
		t.Errorf("Expected stamped version %q, got %q", "v0.1.0", got)
	}
}

func TestResolveVersion_Fallback(t *testing.T) {
	oldVersion := Version
	oldReadBuildInfo := readBuildInfo
	defer func() {
		Version = oldVersion
		readBuildInfo = oldReadBuildInfo
	}()

	Version = "dev"

	t.Run("ValidMainVersion", func(t *testing.T) {
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{
					Version: "v1.0.0",
				},
			}, true
		}
		if got := resolveVersion(); got != "v1.0.0" {
			t.Errorf("Expected 'v1.0.0', got %q", got)
		}
	})

	t.Run("VCSRevisionFallback", func(t *testing.T) {
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{
					Version: "(devel)",
				},
				Settings: []debug.BuildSetting{
					{Key: "vcs.revision", Value: "1234567890abcdef"},
				},
			}, true
		}
		expected := "v0.0.0-dev+1234567"
		if got := resolveVersion(); got != expected {
			t.Errorf("Expected %q, got %q", expected, got)
		}
	})

	t.Run("DevelNoRevisionFallback", func(t *testing.T) {
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return &debug.BuildInfo{
				Main: debug.Module{
					Version: "(devel)",
				},
			}, true
		}
		if got := resolveVersion(); got != "dev" {
			t.Errorf("Expected 'dev', got %q", got)
		}
	})

	t.Run("EmptyVersionFallback", func(t *testing.T) {
		Version = ""
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return nil, false
		}
		if got := resolveVersion(); got != "dev" {
			t.Errorf("Expected 'dev', got %q", got)
		}
	})

	t.Run("NoBuildInfoFallback", func(t *testing.T) {
		Version = "dev"
		readBuildInfo = func() (*debug.BuildInfo, bool) {
			return nil, false
		}
		if got := resolveVersion(); got != "dev" {
			t.Errorf("Expected 'dev', got %q", got)
		}
	})
}
