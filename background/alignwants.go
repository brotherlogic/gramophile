package background

import (
	"context"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

// Ensures everything is in a wantlist - only used when ORIGIN_GRAMOPHILE is set for wants
func (b *BackgroundRunner) AlignWants(ctx context.Context, di discogs.Discogs, c *pb.WantsConfig) error {

	return nil
}
