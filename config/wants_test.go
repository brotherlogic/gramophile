package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestNoWantSpec(t *testing.T) {
	c := &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_GRAMOPHILE, Existing: pb.WantsExisting_EXISTING_UNKNOWN}}

	w := &wants{}
	err := w.Validate(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}
