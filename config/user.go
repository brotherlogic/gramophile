package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type userConfig struct{}

func (*userConfig) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*userConfig) PostProcess(c *pb.GramophileConfig) *pb.GramophileConfig {
	return c
}

func (*userConfig) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetUserConfig().GetUserLevel() == pb.UserConfig_USER_LEVEL_OMNIPOTENT {
		if u.GetUser().GetDiscogsUserId() != 150295 {
			return status.Errorf(codes.FailedPrecondition, "You are not allowed to set the user level to omnipotent")
		}
	}
	return nil
}
