package background

import (
	"context"
	"fmt"
	"time"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshWant(ctx context.Context, d discogs.Discogs, wid int64) error {
	want, err := b.db.GetWant(ctx, d.GetUserId(), wid)
	if err != nil {
		return err
	}

	if want.GetState() == pb.WantState_WANTED {
		_, err := d.AddWant(ctx, wid)
		return err
	}

	return d.DeleteWant(ctx, wid)
}

func (b *BackgroundRunner) SyncWants(ctx context.Context, d discogs.Discogs, user *pb.StoredUser, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, w := range wants {
		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Auth:    user.GetAuth().GetToken(),
				Entry: &pb.QueueElement_RefreshWant{
					RefreshWant: &pb.RefreshWant{
						WantId: w.GetId(),
					}}},
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (b *BackgroundRunner) RefreshWants(ctx context.Context, d discogs.Discogs) error {
	// Look for any wants that have been purchased
	recs, err := b.db.LoadAllRecords(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all records: %w", err)
	}

	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all wants: %w", err)
	}

	for _, want := range wants {
		for _, rec := range recs {
			if want.GetId() == rec.GetRelease().GetId() {
				want.State = pb.WantState_PURCHASED
				err := b.db.SaveWant(ctx, d.GetUserId(), want, "Found purchased record")
				if err != nil {
					return fmt.Errorf("unable to save want: %w", err)
				}
				continue
			}
		}
	}

	return nil
}
