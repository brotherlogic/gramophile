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
	LAST_SALE_UPDATE_FIELD = "LastSaleUpdate"
)

type sales struct{}

func (*sales) PostProcess(c *pb.GramophileConfig) *pb.GramophileConfig {
	return c
}

func (*sales) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	if c.GetSaleConfig().GetMandate() != pb.Mandate_NONE {
		return []*pb.FolderMove{
			{
				Criteria:   &pb.MoveCriteria{HasSaleId: pb.Bool_TRUE, SaleStatus: pbd.SaleStatus_FOR_SALE},
				MoveFolder: "For Sale",
			},
			{
				Criteria:   &pb.MoveCriteria{HasSaleId: pb.Bool_TRUE, SaleStatus: pbd.SaleStatus_SOLD},
				MoveFolder: "Sold",
			},
		}
	}
	return []*pb.FolderMove{}
}

func (*sales) Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	if c.GetSaleConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == LAST_SALE_UPDATE_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("Add a field called '%v'", LAST_SALE_UPDATE_FIELD))
		}
	}

	if c.GetSaleConfig().GetHandlePriceUpdates() != pb.Mandate_NONE {
		if c.GetSaleConfig().GetUpdateFrequencySeconds() == 0 {
			return status.Errorf(codes.FailedPrecondition, fmt.Sprintf("You must set the update frequency field if gramophile is handling price updates"))
		}
	}

	return nil
}
