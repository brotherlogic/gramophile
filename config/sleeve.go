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
	SLEEVE_FIELD = "Sleeve"
)

type sleeve struct{}

func (*sleeve) GetMoves() []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*sleeve) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetSleeveConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == SLEEVE_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", SLEEVE_FIELD))
		}

		if len(c.SleeveConfig.GetAllowedSleeves()) == 0 {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("you must set at least one sleeve"))
		}
	}

	return nil
}
