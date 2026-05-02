package server

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestMovingConfigAccess(t *testing.T) {
	// 1. Verify Standard user cannot access Beta features
	err := CheckAccess(pb.UserConfig_USER_LEVEL_STANDARD, &pb.MovingConfig{})
	if err == nil {
		t.Errorf("Standard user should not have access to MovingConfig")
	}

	// 2. Verify Beta user can access Beta features
	err = CheckAccess(pb.UserConfig_USER_LEVEL_BETA, &pb.MovingConfig{})
	if err != nil {
		t.Errorf("Beta user should have access to MovingConfig: %v", err)
	}

	// 3. Verify Omnipotent user can access Beta features
	err = CheckAccess(pb.UserConfig_USER_LEVEL_OMNIPOTENT, &pb.MovingConfig{})
	if err != nil {
		t.Errorf("Omnipotent user should have access to MovingConfig: %v", err)
	}
}

func TestStandardConfigAccess(t *testing.T) {
	// Verify Standard user can access Standard features
	err := CheckAccess(pb.UserConfig_USER_LEVEL_STANDARD, &pb.CleaningConfig{})
	if err != nil {
		t.Errorf("Standard user should have access to CleaningConfig: %v", err)
	}
}
