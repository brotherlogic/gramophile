package background

import (
	"context"
	"fmt"

	"github.com/brotherlogic/discogs"
)

func (b *BackgroundRunner) RefreshWantlists(ctx context.Context, di discogs.Discogs) error {
	lists, err := b.db.GetWantlists(ctx, di.GetDiscogsUser().GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get wantlists: %w", err)
	}

	for _, list := range lists {
		err = b.processWantlist(ctx, di, list)
		if err != nil {
			return fmt.Errorf("Unable to process wantlist %v -> %w", list.GetName(), err)
		}
	}
}

func (b *BackgroundRunner) processWantlist(ctx context.Context, di discogs.Discogs, list *pb.Wantlist) error {
	records, err := b.db.LoadAllRecords(ctx, di.GetDiscogsUser().GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all records: %w", err)
	}

	changed := false
	for _, entry := range list.GetEntries() {
		if entry.GetStatus() == pb.WantState_WANTED {
			for _, r := range records {
				if r.GetRelease().GetId() == entry.GetId() {
					entry.Status = pb.WantState_PURCHASED
					changed = true
				}
			}
		}
	}

	if changed {
		err := b.db.SaveWantlist(ctx, di.GetDiscogsUser().GetUserId(), list)
		if err != nil {
			return fmt.Errorf("unable to save wantlist: %w", err)
		}

		return b.refreshWantlist(ctx, list)
	}

	return nil
}
