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

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		before int
		after  int
		want   float64
	}{
		{0, 0, 100.0},
		{1, 1, 66.66666666666666},
		{0, 1, 50.0},
		{1, 0, 100.0},
		{9, 0, 100.0},
		{0, 9, 10.0},
		{4, 5, 50.0},
	}

	for _, tt := range tests {
		got := calculatePercentage(tt.before, tt.after)
		if got != tt.want {
			t.Errorf("calculatePercentage(%v, %v) = %v, want %v", tt.before, tt.after, got, tt.want)
		}
	}
}
