package config

import (
	"context"
	"log"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type wants struct{}

func (*wants) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*wants) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	log.Printf("VALID: %v", c)
	if c.GetWantsConfig().GetOrigin() == pb.WantsBasis_WANTS_GRAMOPHILE {
		if c.GetWantsConfig().GetExisting() == pb.WantsExisting_EXISTING_UNKNOWN {
			return status.Errorf(codes.FailedPrecondition, "You must set an existing move")
		}
	}

	return nil
}
