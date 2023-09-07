package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestKeep_Failed(t *testing.T) {
	c := &pb.GramophileConfig{KeepConfig: &pb.KeepConfig{Mandate: pb.Mandate_RECOMMENDED}}

	err := ValidateConfig(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}

func TestKeep_Success(t *testing.T) {
	c := &pb.GramophileConfig{KeepConfig: &pb.KeepConfig{Mandate: pb.Mandate_RECOMMENDED}}

	err := ValidateConfig(context.Background(), []*pbd.Field{&pbd.Field{Name: "Keep", Id: 1}}, c)
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}
}
