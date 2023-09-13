package background

import (
	"context"
	"fmt"
	"log"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RunMoves(ctx context.Context, user *pb.StoredUser, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	moves := user.GetMoves()

	records, err := b.db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unablet to get records: %v", err)
	}

	log.Printf("Running %v moves on %v records", len(moves), len(records))

	return nil
}
