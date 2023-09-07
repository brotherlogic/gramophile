package config

import (
	"context"
	"fmt"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cleaning struct{}

func (*cleaning) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	if c.GetCleaningConfig().GetCreate() == pb.CreateFolders_AUTOMATIC {
		return []*pb.FolderMove{
			{
				Name:         "move_to_cleaning_pile",
				MoveToFolder: "Cleaning Pile",
			},
		}
	}
	return []*pb.FolderMove{}
}

func (*cleaning) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetCleaningConfig().GetCleaning() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == "Cleaned" {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, "Add a field called 'Cleaned'")
		}
	}

	if c.GetCleaningConfig().GetCleaningGapInPlays() > 0 && c.GetCleaningConfig().GetCleaningGapInSeconds() > 0 {
		return fmt.Errorf("You must set one of plays or seconds, not both")
	}
	return nil
}
