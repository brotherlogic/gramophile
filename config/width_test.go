package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestWidthFailedNoField(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{WidthConfig: &pb.WidthConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	w := &width{}
	err := w.Validate(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}

func TestWidthSuccess(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{WidthConfig: &pb.WidthConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	w := &width{}
	err := w.Validate(context.Background(), []*pbd.Field{&pbd.Field{Name: "Width", Id: 1}}, c)
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}
}
