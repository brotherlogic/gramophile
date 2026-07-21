package main

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
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

func TestFormatLocationOutput(t *testing.T) {
	loc := &pb.Location{
		LocationName: "Vinyl Rack",
		Slot:         12,
		Shelf:        "Shelf 1",
		Record:       "The Beatles - Abbey Road",
		Before: []*pb.Context{
			{Record: "Pink Floyd - Dark Side of the Moon"},
		},
		After: []*pb.Context{
			{Record: "Queen - A Night at the Opera"},
		},
	}

	got := formatLocationOutput(loc)
	expected := "The Beatles - Abbey Road is in Vinyl Rack, Slot 12 (67 %):\n\nPink Floyd - Dark Side of the Moon\nThe Beatles - Abbey Road\nQueen - A Night at the Opera\n\n"

	if got != expected {
		t.Errorf("formatLocationOutput mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}

