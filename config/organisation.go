package config

import (
	"context"
	"fmt"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type org struct{}

func (*org) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*org) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {

	// Raise an error if any org relies on width being set
	hasWidthMandate := c.GetWidthConfig().GetMandate() != pb.Mandate_NONE
	for _, org := range c.GetOrganisationConfig().GetOrganisations() {
		if org.GetDensity() == pb.Density_WIDTH && !hasWidthMandate {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("%v requires width mandate", org.GetName()))
		}
	}

	return nil
}
