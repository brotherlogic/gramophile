package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	PURCHASED_PRICE_FIELD    = "Purchase Price"
	PURCHASED_LOCATION_FIELD = "Purcahse Location"
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
			if field.GetName() == PURCHASED_PRICE_FIELD {
				foundp = true
			}
			if field.GetName() == PURCHASED_LOCATION_FIELD {
				foundl = true
			}
		}
		if !foundp {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", PURCHASED_PRICE_FIELD)
		}
		if !foundl {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", PURCHASED_LOCATION_FIELD)
		}
	}

	return nil
}
