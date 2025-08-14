package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type org struct{}

func (*org) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*org) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*org) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {

	// Raise an error if any org relies on width being set
	hasWidthMandate := u.GetConfig().GetWidthConfig().GetMandate() != pb.Mandate_NONE
	for _, org := range u.GetConfig().GetOrganisationConfig().GetOrganisations() {
		if org.GetDensity() == pb.Density_WIDTH && !hasWidthMandate {
			return status.Errorf(codes.FailedPrecondition, "%v requires width mandate", org.GetName())
		}
	}

	return nil
}
