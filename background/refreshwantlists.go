package background

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshWantlists(ctx context.Context, di discogs.Discogs, auth string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	qlog(ctx, "Refreshing Wantlists")
	lists, err := b.db.GetWantlists(ctx, di.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get wantlists: %w", err)
	}
	user, err := b.db.GetUser(ctx, auth)
	if err != nil {
		return err
	}

	// Are we over threshold
	overthreshold := false
	if user.GetConfig().GetWantsListConfig().GetListeningThreshold() > 0 {
		qlog(ctx, "Checking Threshold")
		lp := int32(0)
		for _, org := range user.GetConfig().GetOrganisationConfig().GetOrganisations() {
			if org.GetUse() == pb.OrganisationUse_ORG_USE_LISTENING {
				qlog(ctx, "Threshold for %v", org.GetName())
				ss, err := b.db.GetLatestSnapshot(ctx, user.GetUser().GetDiscogsUserId(), org.GetName())
				if err != nil {
					return err
				}
				qlog(ctx, "Threshold found: %v from %v", len(ss.GetPlacements()), ss.GetHash())
				lp += int32(len(ss.GetPlacements()))
			}
		}

		overthreshold = lp > user.GetConfig().GetWantsListConfig().GetListeningThreshold()
	}

	for _, list := range lists {
		// Reset overthreshold for built lists
		builtList := list.GetName() == "digital_wantlist"

		err = b.processWantlist(ctx, di, user.GetConfig().GetWantsListConfig(), list, auth, overthreshold && !builtList, enqueue)
		if err != nil {
			return fmt.Errorf("Unable to process wantlist %v -> %w", list.GetName(), err)
		}
	}

	user.LastWantlistRefresh = time.Now().UnixNano()
	return b.db.SaveUser(ctx, user)
}

func (b *BackgroundRunner) processWantlist(ctx context.Context, di discogs.Discogs, config *pb.WantslistConfig, list *pb.Wantlist, token string, overthreshold bool, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	qlog(ctx, "Processing %v -> %v with %v (%v)", list.GetName(), list.GetType(), overthreshold, len(list.GetEntries()))

	// Deactivate if set

	// Clear any wants that have an id of zero
	nentries := []*pb.WantlistEntry{}
	for _, entry := range list.GetEntries() {
		if entry.GetId() > 0 || entry.GetMasterId() > 0 {
			nentries = append(nentries, entry)
		}
	}
	list.Entries = nentries

	idMap := make(map[int64]int32)
	records, err := b.db.LoadAllRecords(ctx, di.GetUserId())
	if err != nil {
		return err
	}
	for _, record := range records {
		idMap[record.GetRelease().GetId()] = record.GetRelease().GetRating()
	}

	for _, entry := range list.GetEntries() {
		qlog(ctx, "REFRESH_WANT %v -> %v", list.GetName(), entry)
		// Hard sync from the want
		want, err := b.db.GetWant(ctx, di.GetUserId(), entry.GetId())
		if err != nil {
			if status.Code(err) == codes.NotFound {
				// We need to save this want
				want = &pb.Want{
					Id:    entry.GetId(),
					State: pb.WantState_WANT_UNKNOWN,
				}
				if list.GetVisibility() == pb.WantlistVisibility_INVISIBLE {
					want.State = pb.WantState_HIDDEN
				}
				err = b.db.SaveWant(ctx, di.GetUserId(), want, "Creating from wantlist update")
				if err != nil {
					return nil
				}
			} else {
				return err
			}
		}

		if want.GetId() == entry.GetId() && want.GetState() != entry.GetState() {
			qlog(ctx, "UPDATING WANT STATE %v and %v", want, entry)
			entry.State = want.GetState()
			list.LastPurchaseDate = time.Now().UnixNano()
		}

		if want.GetId() == entry.GetId() && want.GetScore() != entry.GetScore() {
			entry.Score = want.GetScore()
		}
	}

	// Should we deactivate this list
	score := float32(0)
	count := float32(0)
	if config.GetMinCount() > 0 || config.GetMinScore() > 0 {
		for _, entry := range list.GetEntries() {
			qlog(ctx, "Entry: %v", entry)
			if entry.GetState() == pb.WantState_PURCHASED || entry.GetState() == pb.WantState_IN_TRANSIT {
				score += float32(entry.GetScore())
				count++
			} else if entry.GetState() == pb.WantState_RETIRED {
				// If the records retired, we've de-activated the list. So read from the Direct Map
				if val, ok := idMap[entry.GetId()]; ok {
					score += float32(val)
					count++
				}
			}
		}

		qlog(ctx, "Found Score %v / %v", score, count)
		if count >= float32(config.GetMinCount()) {
			list.Active = score/count >= config.GetMinScore()
			qlog(ctx, "Set active: %v (%v vs %v)", list.Active, score/count, config.GetMinScore())
		} else {
			list.Active = true
		}
	}

	_, err = b.refreshWantlist(ctx, di.GetUserId(), list, token, overthreshold, enqueue)
	if err != nil && status.Code(err) != codes.FailedPrecondition {
		return fmt.Errorf("unable to refresh wantlist: %w", err)
	}

	list.LastUpdatedTimestamp = time.Now().UnixNano()

	return b.db.SaveWantlist(ctx, di.GetUserId(), list)
}

func (b *BackgroundRunner) refreshWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, overthreshold bool, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	// If the list is inactive - just set everything to PENDING
	if !list.GetActive() || overthreshold {
		qlog(ctx, "List %v is inactive (%v) or is overthreshold (%v)", list.GetName(), list.GetActive(), overthreshold)

		if !list.GetActive() {
			list.LastChangeDetail = "List is not active, retiring entries"
		} else if overthreshold {
			list.LastChangeDetail = "LP is over threshold, retiring entries"
		}

		for _, entry := range list.GetEntries() {
			if entry.GetState() != pb.WantState_RETIRED {
				entry.State = pb.WantState_RETIRED
				err := b.mergeWant(ctx, userid, &pb.Want{
					Id:    entry.GetId(),
					State: pb.WantState_RETIRED,
				})
				if err != nil {
					return false, err
				}
				_, err = enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					Auth:    token,
					RunDate: time.Now().UnixNano(),
					Entry: &pb.QueueElement_RefreshWant{
						RefreshWant: &pb.RefreshWant{
							Want: &pb.Want{Id: entry.GetId()},
						},
					},
				}})
				if err != nil {
					return false, err
				}
			}
		}
		return true, nil
	}

	nlist, err := b.cleanWantlist(ctx, userid, list)
	if err != nil {
		return false, err
	}

	switch list.GetType() {
	case pb.WantlistType_ONE_BY_ONE:
		return b.refreshOneByOneWantlist(ctx, userid, nlist, token, enqueue)
	case pb.WantlistType_EN_MASSE:
		return b.refreshEnMasseWantlist(ctx, userid, nlist, token, enqueue)
	case pb.WantlistType_DATE_BOUNDED:
		return b.refreshTimedWantlist(ctx, userid, nlist, token, enqueue)
	default:
		qlog(ctx, "Failure to process want list because %v", list.GetType())
		return false, status.Errorf(codes.FailedPrecondition, "%v is not currently processable (%v)", nlist.GetName(), nlist.GetType())
	}
}

// Does cleaning jobs - currently just for any list that has multiple entries from a single source (e.g. the digital wantlist)
func (b *BackgroundRunner) cleanWantlist(ctx context.Context, userid int32, list *pb.Wantlist) (*pb.Wantlist, error) {
	log.Printf("Cleaning Wantlist")
	var deleteIds []int64
	for _, entry := range list.GetEntries() {
		if entry.GetSourceId() > 0 && (entry.GetState() == pb.WantState_PURCHASED || entry.GetState() == pb.WantState_IN_TRANSIT) {
			deleteIds = append(deleteIds, entry.GetSourceId())
		}
	}

	log.Printf("TO DELETE: %v", deleteIds)

	var nentries []*pb.WantlistEntry
	for _, entry := range list.GetEntries() {
		found := false
		for _, did := range deleteIds {
			if entry.GetSourceId() == did {
				found = true
			}
		}
		if !found {
			nentries = append(nentries, entry)
		}
	}
	list.Entries = nentries

	return list, nil
}

func (b *BackgroundRunner) refreshEnMasseWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	updated := false
	qlog(ctx, "HERE HERE Refreshing %v with %v", list.GetName(), len(list.GetEntries()))
	for _, entry := range list.GetEntries() {
		want, err := b.db.GetWant(ctx, userid, entry.GetId())
		if err != nil {
			return false, err
		}

		qlog(ctx, "Tracking: %v", want)
		if want.GetState() != pb.WantState_WANTED &&
			want.GetState() != pb.WantState_PURCHASED &&
			want.GetState() != pb.WantState_IN_TRANSIT {
			want.State = pb.WantState_WANTED
			want.Clean = false
			err = b.db.SaveWant(ctx, userid, want, "Saving from wantlist update")
			_, err = enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
				Auth:    token,
				RunDate: time.Now().Unix(),
				Entry: &pb.QueueElement_RefreshWant{
					RefreshWant: &pb.RefreshWant{
						Want: &pb.Want{
							Id: entry.GetId(),
						},
					},
				},
			}})
			entry.State = pb.WantState_WANTED
			updated = true
		}
	}

	return updated, nil
}

func (b *BackgroundRunner) refreshTimedWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	updated := false
	countWanted := 0
	daysLong := time.Unix(0, list.GetEndDate()).Sub(time.Unix(0, list.GetStartDate())).Hours() / 24
	daysIn := time.Now().Sub(time.Unix(0, list.GetStartDate())).Hours() / 24
	intendedWants := int(math.Ceil(float64(len(list.GetEntries())+1) * daysIn / daysLong))

	qlog(ctx, "TIMED %v %v %v", daysLong, daysIn, intendedWants)

	for _, entry := range list.GetEntries() {
		switch entry.GetState() {
		case pb.WantState_IN_TRANSIT, pb.WantState_WANTED, pb.WantState_PURCHASED:
			countWanted++
		case pb.WantState_PENDING, pb.WantState_WANT_UNKNOWN, pb.WantState_RETIRED:
			if countWanted < intendedWants {
				qlog(ctx, "ENQUEUE %v", entry.GetId())
				_, err := enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					Auth:    token,
					RunDate: time.Now().UnixNano(),
					Entry: &pb.QueueElement_RefreshWant{
						RefreshWant: &pb.RefreshWant{
							Want: &pb.Want{Id: entry.GetId(), State: pb.WantState_WANTED},
						},
					},
				}})
				qlog(ctx, "RESULT %v", err)
				if err != nil {
					return false, err
				}
				updated = true
				entry.State = pb.WantState_WANTED
				countWanted++
			}
		}
	}

	return updated, nil
}

func (b *BackgroundRunner) refreshOneByOneWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	sort.SliceStable(list.GetEntries(), func(i, j int) bool {
		return list.GetEntries()[i].GetIndex() < list.GetEntries()[j].GetIndex()
	})

	foundFirst := false
	for _, entry := range list.GetEntries() {
		qlog(ctx, "Assessing %v in %v [%v]", entry, list.GetName(), foundFirst)
		/*if list.GetActive() {
			err := b.db.SaveWant(ctx, userid, &pb.Want{
				Id:    entry.GetId(),
				State: pb.WantState_PENDING,
			}, "wantlist inactive")
			if err != nil {
				return false, err
			}
			continue
		}*/

		if foundFirst {
			if entry.GetState() != pb.WantState_PENDING {
				b.mergeWant(ctx, userid, &pb.Want{Id: entry.GetId(), State: pb.WantState_PENDING})
				_, err := enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					Auth:    token,
					RunDate: time.Now().UnixNano(),
					Entry: &pb.QueueElement_RefreshWant{
						RefreshWant: &pb.RefreshWant{
							Want: &pb.Want{Id: entry.GetId(), State: pb.WantState_PENDING},
						},
					},
				}})
				if err != nil {
					return false, err
				}
			}
			continue
		}

		qlog(ctx, "Refreshing Queue entry: %v", entry)

		switch entry.GetState() {
		case pb.WantState_IN_TRANSIT:
			foundFirst = true
		case pb.WantState_WANTED:
			foundFirst = true
			if list.GetVisibility() == pb.WantlistVisibility_INVISIBLE {
				err := b.mergeWant(ctx, userid, &pb.Want{
					Id:    entry.GetId(),
					State: pb.WantState_HIDDEN,
				})
				if err != nil {
					return false, err
				}
				_, err = enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					Auth:    token,
					RunDate: time.Now().UnixNano(),
					Entry: &pb.QueueElement_RefreshWant{
						RefreshWant: &pb.RefreshWant{
							Want: &pb.Want{Id: entry.GetId()},
						},
					},
				}})
				if err != nil {
					return false, err
				}
				continue
			}
		case pb.WantState_PURCHASED:
			continue
		case pb.WantState_PENDING, pb.WantState_RETIRED, pb.WantState_WANT_UNKNOWN:
			foundFirst = true
			state := pb.WantState_WANTED
			if list.GetVisibility() == pb.WantlistVisibility_INVISIBLE {
				state = pb.WantState_HIDDEN
			}
			entry.State = state
			qlog(ctx, "ESETTING ENTRY: %v", entry)
			err := b.mergeWant(ctx, userid, &pb.Want{
				Id:    entry.GetId(),
				State: state,
			})
			if err != nil {
				return false, err
			}
			_, err = enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
				Auth:    token,
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_RefreshWant{
					RefreshWant: &pb.RefreshWant{
						Want: &pb.Want{Id: entry.GetId(), State: entry.GetState()},
					},
				},
			}})
			return true, err
		}
	}

	return false, nil
}

func (b *BackgroundRunner) mergeWant(ctx context.Context, userid int32, want *pb.Want) error {
	val, err := b.db.GetWant(ctx, userid, want.GetId())
	if err != nil {
		if status.Code(err) != codes.NotFound {
			val = want
		} else {
			return err
		}
	}

	if want.State != pb.WantState_HIDDEN {
		val.State = want.State
	}
	if want.State == pb.WantState_HIDDEN {
		if val.State == pb.WantState_PENDING || val.State == pb.WantState_WANTED {
			val.State = want.State
		}
	}
	return b.db.SaveWant(ctx, userid, val, "Updated from refresh wantlist")
}
