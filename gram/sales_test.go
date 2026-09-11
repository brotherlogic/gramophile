package main

import (
	"testing"
)

func TestGetSales(t *testing.T) {
	module := GetSales()
	if module == nil {
		t.Fatalf("GetSales returned nil")
	}
	if module.Command != "sales" {
		t.Errorf("Expected command 'sales', got %v", module.Command)
	}
}
