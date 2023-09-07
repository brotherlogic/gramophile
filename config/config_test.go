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

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, config)
	if err == nil {
		t.Errorf("Config was validated: %v", config)
	}
}
