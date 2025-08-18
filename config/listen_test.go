package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestListen_Failed(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{ListenConfig: &pb.ListenConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}

func TestListen_Succeed(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{ListenConfig: &pb.ListenConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "LastListenDate", Id: 1}}, c)
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}
}
