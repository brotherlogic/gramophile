package config

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestValidation(t *testing.T) {
	config := &pb.GramophileConfig{
		CleaningConfig: &pb.CleaningConfig{
			CleaningGapInSeconds: 5,
			CleaningGapInPlays:   2,
		},
	}

	if err := ValidateConfig(config); err == nil {
		t.Errorf("Config was validated: %v", config)
	}

}
