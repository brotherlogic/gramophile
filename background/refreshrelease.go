package background

import (
	"context"

	"github.com/brotherlogic/discogs"
)

func (b *BackgroundRunner) RefreshRelease(ctx context.Context, iid int64, d discogs.Discogs) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return err
	}

	release, err := d.GetRelease(ctx, record.GetRelease().GetId())
	if err != nil {
		return err
	}

	// Update the release from the discogs pull
	record.GetRelease().ReleaseDate = release.GetReleaseDate()
	return b.db.SaveRecord(ctx, d.GetUserId(), record)
}
