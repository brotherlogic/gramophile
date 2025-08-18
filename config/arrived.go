package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ARRIVED_FIELD = "Arrived"
)

type arrived struct{}

func (*arrived) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*arrived) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{
		{
			ClassifierName: "unlistened",
			Classification: "unlistened",
			Rules: []*pb.ClassificationRule{
				{
					RuleName: "not listened yet",
					Selector: &pb.ClassificationRule_IntSelector{IntSelector: &pb.IntSelector{
						Name:      "last_listen",
						Threshold: 0,
						Comp:      pb.Comparator_COMPARATOR_LESS_THAN_OR_EQUALS,
					}},
				},
				{
					RuleName: "arrived",
					Selector: &pb.ClassificationRule_IntSelector{IntSelector: &pb.IntSelector{
						Name:      "has_arrived",
						Threshold: 0,
						Comp:      pb.Comparator_COMPARATOR_GREATER_THAN,
					}},
				},
			},
		},
	}
}

func (*arrived) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetArrivedConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED {
		found := false
		for _, field := range fields {
			if field.GetName() == ARRIVED_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", ARRIVED_FIELD)
		}
	}

	return nil
}
