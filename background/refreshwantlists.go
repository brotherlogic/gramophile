package background

import (
	"context"
	"fmt"
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

	changed := false
	for _, entry := range list.GetEntries() {
		if entry.GetState() == pb.WantState_WANTED {
			for _, r := range records {
				if r.GetRelease().GetId() == entry.GetId() {
					entry.State = pb.WantState_PURCHASED
					changed = true
				}
			}
		}
	}

	if changed {
		err := b.db.SaveWantlist(ctx, di.GetUserId(), list)
		if err != nil {
			return fmt.Errorf("unable to save wantlist: %w", err)
		}

		return b.refreshWantlist(ctx, di.GetUserId(), list)
	}

	return nil
}

func (b *BackgroundRunner) refreshWantlist(ctx context.Context, userid int32, list *pb.Wantlist) error {

	switch list.GetType() {
	case pb.WantlistType_ONE_BY_ONE:
		return b.refreshOneByOneWantlist(ctx, userid, list)
	default:
		return status.Errorf(codes.FailedPrecondition, "%v is not currently processable, %v", list.GetName(), list.GetType())
	}
}

func (b *BackgroundRunner) refreshOneByOneWantlist(ctx context.Context, userid int32, list *pb.Wantlist) error {
	sort.SliceStable(list.GetEntries(), func(i, j int) bool {
		return list.GetEntries()[i].GetIndex() < list.GetEntries()[j].GetIndex()
	})

	for _, entry := range list.GetEntries() {
		switch entry.GetState() {
		case pb.WantState_WANTED:
			return nil
		case pb.WantState_PURCHASED:
			continue
		case pb.WantState_PENDING:
			return b.db.SaveWant(ctx, userid, &pb.Want{
				Id:    entry.GetId(),
				State: pb.WantState_WANTED,
			})
		}
	}

	return nil
}
