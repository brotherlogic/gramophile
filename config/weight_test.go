package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestWeightFailedNoField(t *testing.T) {
	c := &pb.GramophileConfig{WeightConfig: &pb.WeightConfig{Mandate: pb.Mandate_RECOMMENDED}}

	err := ValidateConfig(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed but did not")
	}
}

func TestWeightSuccess(t *testing.T) {
	c := &pb.GramophileConfig{WeightConfig: &pb.WeightConfig{Mandate: pb.Mandate_RECOMMENDED}}

	err := ValidateConfig(context.Background(), []*pbd.Field{{Name: "Weight", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate weight raised an error: %v", err)
	}
}
