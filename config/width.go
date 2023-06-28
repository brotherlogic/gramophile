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
	WIDTH_FIELD = "LastListenDate"
)

type width struct{}

func (*listen) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetWidthConfig().GetEnabled() {
		found := false
		for _, field := range fields {
			if field.GetName() == WIDTH_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", WIDTH_FIELD))
		}
	}
	return nil
}
