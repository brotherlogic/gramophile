package proto

import (
	"testing"
)

func TestLocationTitle(t *testing.T) {
	loc := &Location{
		Title: "Test Artist - Test Title",
	}

	if loc.GetTitle() != "Test Artist - Test Title" {
		t.Errorf("Expected title 'Test Artist - Test Title', got %v", loc.GetTitle())
	}
}
