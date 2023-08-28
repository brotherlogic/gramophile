package background

import (
	"context"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
)

func (b *BackgroundRunner) RefreshUpdates(ctx context.Context, d discogs.Discogs) error {
	rs, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, iid := range rs {
		r, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
		if err != nil {
			return err
		}
		us, err := b.db.GetUpdates(ctx, d.GetUserId(), r)

		for _, update := range us {
			update.Explanation = db.ResolveDiff(update)
			b.db.SaveUpdate(ctx, d.GetUserId(), r, update)
		}
	}

	return nil
}
