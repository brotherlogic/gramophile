package proto

import (
	"testing"
)

func TestLocationRecord(t *testing.T) {
	loc := &Location{
		Record: "Test Artist - Test Title",
	}

	if loc.GetRecord() != "Test Artist - Test Title" {
		t.Errorf("Expected record 'Test Artist - Test Title', got %v", loc.GetRecord())
	}
}
