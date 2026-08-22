package main

import (
	"os"
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
