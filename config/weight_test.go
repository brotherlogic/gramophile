package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestWeightFailedNoField(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{WeightConfig: &pb.WeightConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed but did not")
	}
}

func TestWeightSuccess(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{WeightConfig: &pb.WeightConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Weight", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate weight raised an error: %v", err)
	}
}
