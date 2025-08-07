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
	qlog(ctx, "Refhresing single want: %v", want)
	user, err := b.db.GetUser(ctx, authToken)
	if err != nil {
		return err
	}

	// If this want has no associated wantlist, set it to RETIRED
	if len(want.GetFromWantlist()) == 0 {
		//want.IntendedState = pb.WantState_RETIRED
	}

	var storedWant *pb.Want
	if want.GetMasterId() == 0 {
		storedWant, err = b.db.GetWant(ctx, user.GetUser().GetDiscogsUserId(), want.GetId())
		log.Printf("From %v to %v", want, storedWant)
		if err != nil {
			return err
		}
	} else {
		storedWant, err = b.db.GetMasterWant(ctx, user.GetUser().GetDiscogsUserId(), want.GetMasterId())
		if err != nil {
			return err
		}
	}

	changed, err := b.RefreshWantInternal(ctx, d, storedWant, authToken, enqueue)
	if err != nil {
		return err
	}
	log.Printf("CHANGED %v -> %v", storedWant, changed)

	// Update any wantlist entry that contains this want
	if changed {
		lists, err := b.db.GetWantlists(ctx, user.GetUser().GetDiscogsUserId())
		if err != nil {
			return err
		}
		for _, list := range lists {
			updated := false

			for _, entry := range list.GetEntries() {
				if entry.GetId() == storedWant.GetId() {
					entry.State = storedWant.GetState()
					updated = true
				}
				if updated {
					err = b.db.SaveWantlist(ctx, user.GetUser().GetDiscogsUserId(), list)
					if err != nil {
						return err
					}
				}
			}

		}
	}
	storedWant.Clean = true
	log.Printf("STORED: %v -> %v", storedWant, want)
	return b.db.SaveWant(ctx, user.GetUser().GetDiscogsUserId(), storedWant, "Storing from refresh")
}

func (b *BackgroundRunner) RefreshWantInternal(ctx context.Context, d discogs.Discogs, want *pb.Want, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	qlog(ctx, "Refreshing the want %v", want)
	// If the want is already in the intended state, do nothing
	if want.GetIntendedState() == want.GetState() {
		qlog(ctx, "Not updating want since state balances")
		return false, nil
	}

	log.Printf("Refreshing: %v", want)
	if want.GetIntendedState() == pb.WantState_WANTED {
		if want.GetMasterId() > 0 && want.GetId() == 0 {
			return true, b.handleMasterWant(ctx, d, want, authToken, enqueue)
		}
		_, err := d.AddWant(ctx, want.GetId())
		if err != nil {
			return true, err
		}
		want.State = want.GetIntendedState()
		return true, b.db.SaveWant(ctx, d.GetUserId(), want, "Adding want")
	}

	if want.GetMasterId() > 0 {
		return true, status.Errorf(codes.Internal, "Unable to delete master id currently")
	}
	qlog(ctx, "Deleting want")
	err := d.DeleteWant(ctx, want.GetId())

	// Not Found here is fine if we never wanted this thing in the first place
	if err != nil && status.Code(err) != codes.NotFound {
		return true, err
	}
	want.State = want.GetIntendedState()
	return true, b.db.SaveWant(ctx, d.GetUserId(), want, "Deleting want")
}

func (b *BackgroundRunner) RefreshWants(ctx context.Context, d discogs.Discogs, auth string, enqueue func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	// Look for any wants that have been purchased
	recs, err := b.db.LoadAllRecords(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all records: %w", err)
	}

	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all wants: %w", err)
	}

	log.Printf("Found %v wants and %v records", len(wants), len(recs))

	for _, want := range wants {
		found := false
		for _, rec := range recs {
			if want.GetId() == rec.GetRelease().GetId() {
				found = true
				log.Printf("Refreshing Want %v -> %v", want, rec)
				//want.IntendedState = pb.WantState_IN_TRANSIT
				if rec.GetArrived() > 0 {
					//want.IntendedState = pb.WantState_PURCHASED
				}
				if rec.GetRelease().GetRating() > 0 {
					want.Score = rec.GetRelease().GetRating()
				}
				want.Clean = false
				err := b.db.SaveWant(ctx, d.GetUserId(), want, "Found purchased record")
				if err != nil {
					return fmt.Errorf("unable to save want: %w", err)
				}
				enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						Auth:    auth,
						RunDate: time.Now().UnixNano(),
						Entry: &pb.QueueElement_RefreshWant{
							RefreshWant: &pb.RefreshWant{Want: &pb.Want{Id: want.GetId()}},
						},
					},
				})
				continue
			}
		}

		// This is wrong - reset this
		if !found && (want.GetState() == pb.WantState_IN_TRANSIT || want.GetState() == pb.WantState_PURCHASED) {
			//want.IntendedState = pb.WantState_WANT_UNKNOWN
			err = b.db.SaveWant(ctx, d.GetUserId(), want, "Mislabelled purchase")
			if err != nil {
				return fmt.Errorf("unable to save want: %v", err)
			}
		}
	}

	return nil
}
