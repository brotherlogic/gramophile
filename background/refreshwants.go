package background

import (
	"context"
	"fmt"
	"time"

	"github.com/brotherlogic/discogs"
	dpb "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) AddWant(ctx context.Context, wid int64, d discogs.Discogs) error {
	_, err := d.AddWant(ctx, wid)
	if err != nil {
		return err
	}

	return b.db.SaveWant(ctx, d.GetUserId(), &pb.Want{
		Id: wid,
	}, "Adding want from background task")
}

func wfilter(filter *pb.WantFilter, release *dpb.Release) bool {
	for _, ef := range filter.GetExcludeFormats() {
		for _, f := range release.GetFormats() {
			if f.GetName() == ef {
				return false
			}
		}
	}

	if len(filter.GetFormats()) == 0 {
		return true
	}

	found := false
	for _, af := range filter.GetFormats() {
		for _, f := range release.GetFormats() {
			if f.GetName() == af {
				found = true
			}
		}
	}
	return found
}

func AddMasterWant(ctx context.Context, wid int64, filter *pb.WantFilter, d discogs.Discogs, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	release, err := d.GetRelease(ctx, wid)
	if err != nil {
		return err
	}

	if wfilter(filter, release) {
		enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Auth: authToken,
				Entry: &pb.QueueElement_AddWant{
					AddWant: &pb.AddWant{
						Id:       release.GetId(),
						MasterId: release.GetMasterId(),
					},
				},
			},
		})
	}
}

func (b *BackgroundRunner) handleMasterWant(ctx context.Context, d discogs.Discogs, want *pb.Want, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	master, err := d.GetMasterReleases(ctx, want.GetMasterId(), 1, dpb.MasterSort_BY_YEAR)
	if err != nil {
		return err
	}

	for _, pwant := range master {
		enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Auth: authToken,
				Entry: &pb.QueueElement_AddWant{
					AddWant: &pb.AddWant{Id: pwant, Masterid: want.GetMasterId(), Filter: want.GetMasterFilter()},
				},
			}})
	}
}

func (b *BackgroundRunner) RefreshWant(ctx context.Context, d discogs.Discogs, wid int64, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	want, err := b.db.GetWant(ctx, d.GetUserId(), wid)
	if err != nil {
		return err
	}

	if want.GetState() == pb.WantState_WANTED {
		if want.GetMasterId() > 0 {
			return b.handleMasterWant(ctx, d, want, authToken, enqueue)
		}
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
