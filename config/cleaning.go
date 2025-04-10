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
	CLEANED_FIELD_NAME = "Cleaned"
)

type cleaning struct{}

func (*cleaning) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*cleaning) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{
		{
			ClassifierName: "Uncleaned",
			Classification: "uncleaned",
			Rules: []*pb.ClassificationRule{
				{
					RuleName: "needs_clean",
					Selector: &pb.ClassificationRule_IntSelector{IntSelector: &pb.IntSelector{
						Name:      "last_cleaned",
						Threshold: 0,
						Comp:      pb.Comparator_COMPARATOR_LESS_THAN_OR_EQUALS,
					},
					},
				},
			},
		},
	}
}

func (*cleaning) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetCleaningConfig().GetCleaning() != pb.Mandate_NONE {
		found := false
		for _, field := range fields {
			if field.GetName() == CLEANED_FIELD_NAME {
				found = true
			}
		}
		if !found {
			return status.Errorf(codes.FailedPrecondition, "Add a field called 'Cleaned'")
		}
	}

	if u.GetConfig().GetCleaningConfig().GetCleaningGapInPlays() > 0 && u.GetConfig().GetCleaningConfig().GetCleaningGapInSeconds() > 0 {
		return fmt.Errorf("You must set one of plays or seconds, not both")
	}
	return nil
}
