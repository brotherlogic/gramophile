package background

import (
	"context"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshWantlists(ctx context.Context, di discogs.Discogs, auth string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	lists, err := b.db.GetWantlists(ctx, di.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get wantlists: %w", err)
	}

	for _, list := range lists {
		err = b.processWantlist(ctx, di, list, auth, enqueue)
		if err != nil {
			return fmt.Errorf("Unable to process wantlist %v -> %w", list.GetName(), err)
		}
	}

	return nil
}

func (b *BackgroundRunner) processWantlist(ctx context.Context, di discogs.Discogs, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("Processing %v -> %v", list.GetName(), list.GetType())

	changed := false
	for _, entry := range list.GetEntries() {
		log.Printf("REFRESH %v -> %v", list.GetName(), entry)
		// Hard sync from the want
		want, err := b.db.GetWant(ctx, di.GetUserId(), entry.GetId())
		if err != nil {
			return err
		}

		if want.GetId() == entry.GetId() && want.GetState() != entry.GetState() {
			log.Printf("HERE %v and %v", want, entry)
			entry.State = want.GetState()
			changed = true
			list.LastPurchaseDate = time.Now().UnixNano()
		}
	}

	rchanged, err := b.refreshWantlist(ctx, di.GetUserId(), list, token, enqueue)
	if err != nil && status.Code(err) != codes.FailedPrecondition {
		return fmt.Errorf("unable to refresh wantlist: %w", err)
	}

	if changed || rchanged {
		log.Printf("List has changed: %v", list)
		err := b.db.SaveWantlist(ctx, di.GetUserId(), list)
		if err != nil {
			return fmt.Errorf("unable to save wantlist: %w", err)
		}
	}

	return nil
}

func (b *BackgroundRunner) refreshWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	switch list.GetType() {
	case pb.WantlistType_ONE_BY_ONE:
		return b.refreshOneByOneWantlist(ctx, userid, list, token, enqueue)
	case pb.WantlistType_EN_MASSE:
		return b.refreshEnMasseWantlist(ctx, userid, list, token, enqueue)
	default:
		log.Printf("Failure to process want list because %v", list.GetType())
		return false, status.Errorf(codes.FailedPrecondition, "%v is not currently processable (%v)", list.GetName(), list.GetType())
	}
}

func (b *BackgroundRunner) refreshEnMasseWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	updated := false
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

func (b *BackgroundRunner) refreshOneByOneWantlist(ctx context.Context, userid int32, list *pb.Wantlist, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) (bool, error) {
	sort.SliceStable(list.GetEntries(), func(i, j int) bool {
		return list.GetEntries()[i].GetIndex() < list.GetEntries()[j].GetIndex()
	})

	for _, entry := range list.GetEntries() {
		if list.GetActive() {
			err := b.db.SaveWant(ctx, userid, &pb.Want{
				Id:    entry.GetId(),
				State: pb.WantState_PENDING,
			}, "wantlist inactive")
			if err != nil {
				return false, err
			}
			continue
		}

		log.Printf("Refreshing Queue entry: %v", entry)
		switch entry.GetState() {
		case pb.WantState_WANTED:
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
				return true, err
			}

			return false, nil
		case pb.WantState_PURCHASED:
			continue
		case pb.WantState_PENDING, pb.WantState_RETIRED, pb.WantState_WANT_UNKNOWN:
			state := pb.WantState_WANTED
			if list.GetVisibility() == pb.WantlistVisibility_INVISIBLE {
				state = pb.WantState_HIDDEN
			}
			entry.State = state
			log.Printf("ESETTING ENTRY: %v", entry)
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
