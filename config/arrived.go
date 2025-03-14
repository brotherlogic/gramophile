package config

import (
	"context"
	"fmt"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ARRIVED_FIELD = "Arrived"
)

type arrived struct{}

func (*arrived) PostProcess(c *pb.GramophileConfig) *pb.GramophileConfig {
	return c
}

func (*arrived) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	if c.GetArrivedConfig().GetMandate() != pb.Mandate_NONE {
		return []*pb.FolderMove{
			{
				Name:       "Move_To_Listening_Pile_Once_Arrived",
				Origin:     pb.Create_AUTOMATIC,
				MoveFolder: "Listening Pile",
				Criteria: &pb.MoveCriteria{ // Move if we've marked as arrived, but we've not listened to it yet
					Listened: pb.Bool_FALSE,
					Arrived:  pb.Bool_TRUE,
				},
			},
		}
	}
	return []*pb.FolderMove{}
}

func (*arrived) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetArrivedConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == ARRIVED_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", ARRIVED_FIELD))
		}
	}

	return nil
}
