package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestValidation(t *testing.T) {
	config := &pb.GramophileConfig{
		CleaningConfig: &pb.CleaningConfig{
			CleaningGapInSeconds: 5,
			CleaningGapInPlays:   2,
		},
	}

	if err := ValidateConfig(context.Background(), []*pbd.Field{}, config); err == nil {
		t.Errorf("Config was validated: %v", config)
	}

}
