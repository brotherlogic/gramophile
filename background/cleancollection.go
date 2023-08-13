package background

import (
	"context"

	"github.com/brotherlogic/discogs"
)

func (b *BackgroundRunner) CleanCollection(ctx context.Context, d discogs.Discogs, refreshId int64) error {
	records, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, r := range records {
		record, err := b.db.GetRecord(ctx, d.GetUserId(), r)
		if err != nil {
			return err
		}

		if record.GetRefreshId() != refreshId {
			b.db.DeleteRecord(ctx, d.GetUserId(), r)
		}
	}

	return nil
}
