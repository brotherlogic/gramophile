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
	WEIGHT_FIELD = "Weight"
)

type weight struct{}

func (*weight) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetWeightConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == WEIGHT_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", WEIGHT_FIELD))
		}
	}
	return nil
}
