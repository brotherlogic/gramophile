package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSetOmnipotent_Success(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{UserConfig: &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_OMNIPOTENT}}, User: &pbd.User{DiscogsUserId: 150295}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err != nil {
		t.Errorf("Unable to set user level")
	}
}

func TestSetOmnipotent_Failure(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{UserConfig: &pb.UserConfig{UserLevel: pb.UserConfig_USER_LEVEL_OMNIPOTENT}}, User: &pbd.User{DiscogsUserId: 150296}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("I was able to set user level")
	}
}
