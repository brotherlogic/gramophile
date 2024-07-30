package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	dpb "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func wfilter(filter *pb.WantFilter, release *dpb.Release) bool {
	log.Printf("FILTER: %v", filter)
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

func (b *BackgroundRunner) AddMasterWant(ctx context.Context, d discogs.Discogs, want *pb.Want) error {
	// Load the record
	record, err := d.GetRelease(ctx, want.GetId())
	if err != nil {
		return err
	}

	if wfilter(want.GetMasterFilter(), record) {
		return b.db.SaveWant(ctx, d.GetUserId(), want, "Adding from master")
	}
	return nil
}

func (b *BackgroundRunner) handleMasterWant(ctx context.Context, d discogs.Discogs, want *pb.Want, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	master, err := d.GetMasterReleases(ctx, want.GetMasterId(), 1, dpb.MasterSort_BY_YEAR)
	log.Printf("MASTERS: %v %v", master, err)
	if err != nil {
		return err
	}

	for _, pwant := range master {
		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Auth:    authToken,
				RunDate: time.Now().UnixNano(),
				Entry:   &pb.QueueElement_AddMasterWant{AddMasterWant: &pb.AddMasterWant{Want: &pb.Want{Id: pwant.GetId(), MasterId: want.GetMasterId(), MasterFilter: want.GetMasterFilter()}}},
			}})
	}

	// resync the wants if we added anything
	if len(master) > 0 {
		_, err := enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Auth:    authToken,
				RunDate: time.Now().UnixNano(),
				Entry:   &pb.QueueElement_SyncWants{},
			},
		})
		return err
	}

	return nil
}

func (b *BackgroundRunner) RefreshWant(ctx context.Context, d discogs.Discogs, want *pb.Want, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	user, err := b.db.GetUser(ctx, authToken)
	if err != nil {
		return err
	}

	storedWant, err := b.db.GetWant(ctx, user.GetUser().GetDiscogsUserId(), want.GetId())
	if err != nil {
		return err
	}

	err = b.RefreshWantInternal(ctx, d, want, authToken, enqueue)
	if err != nil {
		return err
	}

	storedWant.Clean = true
	return b.db.SaveWant(ctx, user.GetUser().GetDiscogsUserId(), storedWant, "Storing from refresh")
}

func (b *BackgroundRunner) RefreshWantInternal(ctx context.Context, d discogs.Discogs, want *pb.Want, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("Refreshing: %v", want)
	if want.GetState() == pb.WantState_WANTED {
		if want.GetMasterId() > 0 && want.GetId() == 0 {
			return b.handleMasterWant(ctx, d, want, authToken, enqueue)
		}
		_, err := d.AddWant(ctx, want.GetId())
		return err
	}

	if want.GetMasterId() > 0 {
		return status.Errorf(codes.Internal, "Unable to delete master id currently")
	}
	return d.DeleteWant(ctx, want.GetId())
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
				want.State = pb.WantState_IN_TRANSIT
				if rec.GetArrived() > 0 {
					want.State = pb.WantState_PURCHASED
				}
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
