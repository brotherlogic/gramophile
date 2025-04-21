package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
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
	return nil
}
