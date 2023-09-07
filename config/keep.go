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
	KEEP_FIELD = "Keep"
)

type keep struct{}

func (*keep) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*keep) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetKeepConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == KEEP_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", KEEP_FIELD))
		}
	}
	return nil
}
