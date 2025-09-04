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
		CleaningConfig: &pb.CleaningConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		UserConfig:     &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_BETA},
	}
	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, &pb.StoredUser{Config: c})
	if err != nil {
		t.Errorf("Error reseting config: %v", err)
	}

	if c.GetCleaningConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED {
		t.Errorf("Config was not reset")
	}
}

func TestClearOnStandard_NotSet(t *testing.T) {
	c := &pb.GramophileConfig{
		UserConfig:     &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_STANDARD},
		CleaningConfig: &pb.CleaningConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		WidthConfig: &pb.WidthConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WeightConfig: &pb.WeightConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		SleeveConfig: &pb.SleeveConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ListenConfig: &pb.ListenConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		GoalFolderConfig: &pb.GoalFolderConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ArrivedConfig: &pb.ArrivedConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		SaleConfig: &pb.SaleConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		KeepConfig: &pb.KeepConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		PrintMoveConfig: &pb.PrintMoveConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		MintUpConfig: &pb.MintUpConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WantsListConfig: &pb.WantslistConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WantsConfig: &pb.WantsConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		ScoreConfig: &pb.ScoreConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ClassificationConfig: &pb.ClassificationConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		MovingConfig: &pb.MovingConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		AddConfig: &pb.AddConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		OrganisationConfig: &pb.OrganisationConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
	}
	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{
		Name: "Width", Id: 1},
	}, &pb.StoredUser{Config: c})
	if err != nil {
		t.Errorf("Error reseting config: %v", err)
	}

	// Supported on default:
	if c.GetWidthConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED {
		t.Errorf("Width Config was not reset")
	}

	// Not supported on default
	if c.GetCleaningConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED {
		t.Errorf("Cleaning Config was not reset")
	}

}

func TestClearOnBeta_NotSet(t *testing.T) {
	c := &pb.GramophileConfig{
		UserConfig:     &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_BETA},
		CleaningConfig: &pb.CleaningConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		WidthConfig: &pb.WidthConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WeightConfig: &pb.WeightConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		SleeveConfig: &pb.SleeveConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ListenConfig: &pb.ListenConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		GoalFolderConfig: &pb.GoalFolderConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ArrivedConfig: &pb.ArrivedConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		SaleConfig: &pb.SaleConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		KeepConfig: &pb.KeepConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		PrintMoveConfig: &pb.PrintMoveConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		MintUpConfig: &pb.MintUpConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WantsListConfig: &pb.WantslistConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		WantsConfig: &pb.WantsConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		ScoreConfig: &pb.ScoreConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		ClassificationConfig: &pb.ClassificationConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		MovingConfig: &pb.MovingConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		AddConfig: &pb.AddConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
		OrganisationConfig: &pb.OrganisationConfig{
			Enabled: pb.Enabled_ENABLED_ENABLED,
		},
	}
	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{
		Name: "Width", Id: 1},
	}, &pb.StoredUser{Config: c})
	if err != nil {
		t.Errorf("Error reseting config: %v", err)
	}

	// Supported on default:
	if c.GetWidthConfig().GetEnabled() != pb.Enabled_ENABLED_ENABLED {
		t.Errorf("Width Config was cleared")
	}

	// Not supported on default
	if c.GetCleaningConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED {
		t.Errorf("Cleaning Config was not reset")
	}

}
