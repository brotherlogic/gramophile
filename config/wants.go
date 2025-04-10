package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type wants struct{}

func (*wants) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*wants) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*wants) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetWantsConfig().GetOrigin() == pb.WantsBasis_WANTS_GRAMOPHILE {
		if u.GetConfig().GetWantsConfig().GetExisting() == pb.WantsExisting_EXISTING_UNKNOWN {
			return status.Errorf(codes.FailedPrecondition, "You must set an existing move")
		}
	}

	return nil
}
