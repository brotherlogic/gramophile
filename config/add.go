package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	PURCHASED_PRICE    = "Purchase Price"
	PURCHASED_LOCATION = "Purcahse Location"
)

type add struct{}

func (*add) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*add) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*add) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetAddConfig().GetAllowAdds() != pb.Mandate_NONE {
		foundp := false
		foundl := false
		for _, field := range fields {
			if field.GetName() == PURCHASED_PRICE {
				foundp = true
			}
			if field.GetName() == PURCHASED_LOCATION {
				foundl = true
			}
		}
		if !foundp {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", PURCHASED_PRICE)
		}
		if !foundl {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", PURCHASED_LOCATION)
		}
	}

	return nil
}
