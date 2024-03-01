package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	RefreshReleasePeriod      = time.Hour * 24          // Once a week
	RefreshReleaseDatesPeriod = time.Hour * 24 * 7 * 30 // Once a month
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

		if time.Since(time.Unix(0, rec.GetLastUpdateTime())) > RefreshReleasePeriod {
			_, err = enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano(),
					Auth:    authToken,
					Entry: &pb.QueueElement_RefreshRelease{
						RefreshRelease: &pb.RefreshRelease{
							Iid:       id,
							Intention: "from-refresh-collection",
						}}},
			})
			if err == nil {
				log.Printf("Refreshing %v", id)
			}

			// If the refresh is already in the queue, then that's fine
			if err != nil && status.Code(err) != codes.AlreadyExists {
				return err
			}
			if time.Since(time.Unix(0, rec.GetLastEarliestReleaseUpdate())) > RefreshReleaseDatesPeriod {
				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate: time.Now().UnixNano(),
						Auth:    authToken,
						Entry: &pb.QueueElement_RefreshEarliestReleaseDates{
							RefreshEarliestReleaseDates: &pb.RefreshEarliestReleaseDates{
								Iid:      id,
								MasterId: rec.GetRelease().GetMasterId(),
							}}},
				})
				if err != nil && status.Code(err) != codes.AlreadyExists {
					return err
				}
			}
		} else {
			skipped++
		}
	}

	log.Printf("Skipped %v releases", skipped)

	return nil
}
