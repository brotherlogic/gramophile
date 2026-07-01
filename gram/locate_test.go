package main

import (
	"testing"
)

func TestGetLocate(t *testing.T) {
	module := GetLocate()
	if module == nil {
		t.Fatalf("GetLocate returned nil")
	}
	if module.Command != "locate" {
		t.Errorf("Expected command 'locate', got %v", module.Command)
	}
}
