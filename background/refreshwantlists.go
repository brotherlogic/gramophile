package background

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshWantlists(ctx context.Context, di discogs.Discogs, auth string) error {
	lists, err := b.db.GetWantlists(ctx, di.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get wantlists: %w", err)
	}

	for _, list := range lists {
		err = b.processWantlist(ctx, di, list)
		if err != nil {
			return fmt.Errorf("Unable to process wantlist %v -> %w", list.GetName(), err)
		}
	}

	return nil
}

func (b *BackgroundRunner) processWantlist(ctx context.Context, di discogs.Discogs, list *pb.Wantlist) error {
	records, err := b.db.LoadAllRecords(ctx, di.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all records: %w", err)
	}

	log.Printf("Found records: %v", records)

	changed := false
	for _, entry := range list.GetEntries() {
		log.Printf("Processing wantlist entry: %v", entry)
		if entry.GetState() == pb.WantState_WANTED {
			log.Printf("STATE matches")
			for _, r := range records {
				if r.GetRelease().GetId() == entry.GetId() {
					log.Printf("Found match %v", r)
					entry.State = pb.WantState_PURCHASED
					changed = true
					log.Printf("Resovled: %v", entry)
				}
			}
		}
	}

	rchanged, err := b.refreshWantlist(ctx, di.GetUserId(), list)
	if err != nil {
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

func (b *BackgroundRunner) refreshWantlist(ctx context.Context, userid int32, list *pb.Wantlist) (bool, error) {
	switch list.GetType() {
	case pb.WantlistType_ONE_BY_ONE:
		return b.refreshOneByOneWantlist(ctx, userid, list)
	default:
		return false, status.Errorf(codes.FailedPrecondition, "%v is not currently processable (%v)", list.GetName(), list.GetType())
	}
}

func (b *BackgroundRunner) refreshOneByOneWantlist(ctx context.Context, userid int32, list *pb.Wantlist) (bool, error) {
	sort.SliceStable(list.GetEntries(), func(i, j int) bool {
		return list.GetEntries()[i].GetIndex() < list.GetEntries()[j].GetIndex()
	})

	for _, entry := range list.GetEntries() {
		log.Printf("Refreshing entry: %v", entry)
		switch entry.GetState() {
		case pb.WantState_WANTED:
			return false, nil
		case pb.WantState_PURCHASED:
			continue
		case pb.WantState_PENDING:
			state := pb.WantState_WANTED
			if list.GetVisibility() == pb.WantlistVisibility_INVISIBLE {
				state = pb.WantState_HIDDEN
			}
			entry.State = state
			log.Printf("ENTRY: %v", entry)
			return true, b.mergeWant(ctx, userid, &pb.Want{
				Id:    entry.GetId(),
				State: state,
			})
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

}
