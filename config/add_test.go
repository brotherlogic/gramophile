package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestAddConfigEnabled_NoPrice(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{AddConfig: &pb.AddConfig{
		Enabled: pb.Enabled_ENABLED_ENABLED}}}

	af := &add{}
	err := af.Validate(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}

func TestAddConfigEnabled_Success(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{AddConfig: &pb.AddConfig{
		Enabled: pb.Enabled_ENABLED_ENABLED}}}

	af := &add{}
	err := af.Validate(context.Background(), []*pbd.Field{
		{
			Name: "Purchase Price",
		},
		{
			Name: "Purchase Location",
		},
	}, c)
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}
}
