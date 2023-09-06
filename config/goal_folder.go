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
	GOAL_FOLDER_FIELD = "Goal Folder"
)

type goalFolder struct{}

func (*goalFolder) GetMoves() []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*goalFolder) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetGoalFolderConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == GOAL_FOLDER_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", GOAL_FOLDER_FIELD))
		}
	}
	return nil
}
