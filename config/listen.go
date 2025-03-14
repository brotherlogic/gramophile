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
	LISTEN_FIELD = "LastListenDate"
)

type listen struct{}

func (*listen) PostProcess(c *pb.GramophileConfig) *pb.GramophileConfig {
	return c
}

func (*listen) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*listen) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetListenConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == LISTEN_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", LISTEN_FIELD))
		}
	}
	return nil
}
