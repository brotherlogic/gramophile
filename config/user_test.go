package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSetOmnipotent_Success(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{UserConfig: &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_OMNIPOTENT}}, User: &pbd.User{DiscogsUserId: 150295}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err != nil {
		t.Errorf("Unable to set user level")
	}
}

func TestSetOmnipotent_Failure(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{UserConfig: &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_OMNIPOTENT}}, User: &pbd.User{DiscogsUserId: 150296}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("I was able to set user level")
	}
}

func TestClearOnBeta(t *testing.T) {
	c := &pb.GramophileConfig{
		CleaningConfig: &pb.CleaningConfig{Cleaning: pb.Mandate_REQUIRED},
		UserConfig:     &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_BETA},
	}
	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, &pb.StoredUser{Config: c})
	if err != nil {
		t.Errorf("Error reseting config: %v", err)
	}

	if c.GetCleaningConfig().GetCleaning() == pb.Mandate_REQUIRED {
		t.Errorf("Config was not reset")
	}
}

func TestClearOnBeta_NotSet(t *testing.T) {
	c := &pb.GramophileConfig{
		UserConfig: &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_BETA},
	}
	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, &pb.StoredUser{Config: c})
	if err != nil {
		t.Errorf("Error reseting config: %v", err)
	}

	if c.GetCleaningConfig().GetCleaning() == pb.Mandate_REQUIRED {
		t.Errorf("Config was not reset")
	}
}
