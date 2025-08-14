package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	LAST_SALE_UPDATE_FIELD = "LastSaleUpdate"
)

type sales struct{}

func (*sales) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{
		{
			ClassifierName: "For Sale",
			Classification: "for_sale",
			Rules: []*pb.ClassificationRule{
				{
					RuleName: "has_sale_id",
					Selector: &pb.ClassificationRule_IntSelector{IntSelector: &pb.IntSelector{
						Threshold: 0,
						Comp:      pb.Comparator_COMPARATOR_GREATER_THAN,
						Name:      "sale_id",
					}},
				},
				{
					RuleName: "is_for_sale",
					/*Selector: &pb.ClassificationRule_EnumSelector{EnumSelector: &pb.EnumSelector{
						Name:  "release.sale_state",
						Value: "FOR_SALE",
					},
					},*/
				},
			},
		}}
}

func (*sales) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
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

func (*sales) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetSaleConfig().GetMandate() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == LAST_SALE_UPDATE_FIELD {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, "Add a field called '%v'", LAST_SALE_UPDATE_FIELD)
		}
	}

	if u.GetConfig().GetSaleConfig().GetHandlePriceUpdates() != pb.Mandate_NONE {
		if u.GetConfig().GetSaleConfig().GetUpdateFrequencySeconds() == 0 {
			return status.Errorf(codes.FailedPrecondition, "You must set the update frequency field if gramophile is handling price updates")
		}
	}

	return nil
}
