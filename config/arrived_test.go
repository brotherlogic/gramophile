package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestArrived_FailedNoField(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{ArrivedConfig: &pb.ArrivedConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed but did not (%v)", c)
	}
}

func TestArrived_Success(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{ArrivedConfig: &pb.ArrivedConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate arrived raised an error: %v", err)
	}
}
