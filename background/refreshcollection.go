package background

import (
	"context"
	"fmt"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	RefreshReleasePeriod      = time.Hour * 24 * 7      // Once a week
	RefreshReleaseDatesPeriod = time.Hour * 24 * 7 * 30 // Once a month
)

var ()

type refreshCollectionHandler struct {
	b *BackgroundRunner
}

func (h *refreshCollectionHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	qlog(ctx, "RefreshCollection -> %v", entry.GetRefreshCollection().GetIntention())
	return h.b.RefreshCollection(ctx, d, entry.GetAuth(), enqueue)
}

func (h *refreshCollectionHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *refreshCollectionHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return fmt.Sprintf("RefreshCollection-%v", entry.GetAuth())
}

func (b *BackgroundRunner) RefreshCollection(ctx context.Context, d discogs.Discogs, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	ids, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get records: %w", err)
	}

	skipped := 0

	qlog(ctx, "Refreshing %v releases", len(ids))
	for _, id := range ids {
		qlog(ctx, "REFRESH: %v", id)
		rec, err := b.db.GetRecord(ctx, d.GetUserId(), id)
		if err != nil {
			return fmt.Errorf("unable to get record %v: %w", id, err)
		}

		if rec.GetHighPrice().GetValue() == 0 || time.Since(time.Unix(0, rec.GetLastUpdateTime())) > RefreshReleasePeriod {
			_, err = enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention: "From refresh collection",
					RunDate:   time.Now().UnixNano(),
					Auth:      authToken,
					Entry: &pb.QueueElement_RefreshRelease{
						RefreshRelease: &pb.RefreshRelease{
							Iid:       id,
							Intention: "from-refresh-collection",
						}}},
			})
			if err == nil {
				qlog(ctx, "Refreshing %v", id)
			}

			// If the refresh is already in the queue, then that's fine
			if err != nil && status.Code(err) != codes.AlreadyExists {
				return err
			}
			if time.Since(time.Unix(0, rec.GetLastEarliestReleaseUpdate())) > RefreshReleaseDatesPeriod {
				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						Intention: "From refresh collection",
						RunDate:   time.Now().UnixNano(),
						Auth:      authToken,
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
			qlog(ctx, "SKIPPING DATE REFRESH %v", id)
			skipped++
		}
	}

	qlog(ctx, "Skipped %v releases", skipped)
	user, err := b.db.GetUser(ctx, authToken)
	if err != nil {
		return fmt.Errorf("unable to get user: %w", err)
	}
	user.LastCollectionCheck = time.Now().UnixNano()
	return b.db.SaveUser(ctx, user)
}
