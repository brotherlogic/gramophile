package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	WEIGHT_FIELD = "Weight"
)

type weight struct{}

func (*weight) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*weight) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*weight) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetWeightConfig().GetEnabled() != pb.Enabled_ENABLED_ENABLED {
		found := false
		for _, field := range fields {
			if field.GetName() == WEIGHT_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", WEIGHT_FIELD)
		}
	}
	return nil
}
