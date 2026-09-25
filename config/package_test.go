package config

import (
	"testing"
)

func TestPackageFieldConstant(t *testing.T) {
	if PACKAGE_FIELD != "Package" {
		t.Errorf("Expected PACKAGE_FIELD to be 'Package', got %v", PACKAGE_FIELD)
	}
}
