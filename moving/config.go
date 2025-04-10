package moving

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Moving struct{}

func (*Moving) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*Moving) Validate(ctx context.Context, fields []*pbd.Field, user *pb.StoredUser) error {

	// Validate that all the move locations actually exist in the org config
	for _, move := range user.GetConfig().GetMovingConfig().GetMoves() {
		found := false
		for _, folder := range user.GetFolders() {
			if folder.GetName() == move.GetFolder() {
				found = true
			}
		}

		if !found {
			return status.Errorf(codes.InvalidArgument, "Could not find %v in organisation config", move.GetFolder())
		}
	}

	return nil
}

func (*Moving) PostProcess(config *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	existing := int32(len(config.GetMovingConfig().GetFormatClassifier().GetFormats()))

	if config.GetMovingConfig().GetEnabled() {
		// Apply our default rules over the top of the existing
		config.GetMovingConfig().GetFormatClassifier().Formats = append(config.GetMovingConfig().GetFormatClassifier().GetFormats(),
			&pb.FormatSelector{
				Format:   "12 Inch",
				Contains: []string{"12\"", "LP"},
				Order:    existing + 1,
			})
		config.GetMovingConfig().GetFormatClassifier().Formats = append(config.GetMovingConfig().GetFormatClassifier().GetFormats(),
			&pb.FormatSelector{
				Format:   "7 Inch",
				Contains: []string{"7\""},
				Order:    existing + 2,
			})
		config.GetMovingConfig().GetFormatClassifier().Formats = append(config.GetMovingConfig().GetFormatClassifier().GetFormats(),
			&pb.FormatSelector{
				Format:   "10 Inch",
				Contains: []string{"10\""},
				Order:    existing + 3,
			})
		config.GetMovingConfig().GetFormatClassifier().Formats = append(config.GetMovingConfig().GetFormatClassifier().GetFormats(),
			&pb.FormatSelector{
				Format:      "CD",
				Description: []string{"CD"},
				Order:       existing + 4,
			})
		config.GetMovingConfig().GetFormatClassifier().Formats = append(config.GetMovingConfig().GetFormatClassifier().GetFormats(),
			&pb.FormatSelector{
				Format:      "Digital",
				Description: []string{"File"},
				Order:       existing + 5,
			})
	}

	return config, nil
}
