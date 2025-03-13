package moving

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

type Moving struct{}

func (*Moving) GetMoves(c *pb.GramophileConfig) []*pb.FolderMove {
	return []*pb.FolderMove{}
}

func (*Moving) Validate(ctx context.Context, fields []*pbd.Field, config *pb.GramophileConfig) error {
	return nil
}

func (*Moving) PostProcess(config *pb.GramophileConfig) *pb.GramophileConfig {
	existing := int32(len(config.GetMovingConfig().GetFormatClassifier().GetFormats()))

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

	return config
}
