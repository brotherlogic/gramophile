package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshCollection(ctx context.Context, d discogs.Discogs, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	ids, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get records: %w", err)
	}

	skipped := 0
	log.Printf("Refreshing %v releases", len(ids))
	for _, id := range ids {
		rec, err := b.db.GetRecord(ctx, d.GetUserId(), id)
		if err != nil {
			return fmt.Errorf("unable to get record %v: %w", id, err)
		}

		if time.Since(time.Unix(rec.GetLastUpdateTime(), 0)) > time.Hour*24 {
			_, err = enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate: time.Now().Unix(),
					Auth:    authToken,
					Entry: &pb.QueueElement_RefreshRelease{
						RefreshRelease: &pb.RefreshRelease{
							Iid: id,
						}}},
			})
			if err != nil {
				return err
			}
		} else {
			skipped++
		}
	}

	log.Printf("Skipped %v releases", skipped)

	return nil
}
