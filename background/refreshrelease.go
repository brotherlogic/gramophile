package background

import (
	"context"
	"fmt"

	"github.com/brotherlogic/discogs"
)

func (b *BackgroundRunner) RefreshRelease(ctx context.Context, iid int64, d discogs.Discogs) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	release, err := d.GetRelease(ctx, record.GetRelease().GetId())
	if err != nil {
		return fmt.Errorf("unable to get release %v from discogs: %w", record.GetRelease().GetId(), err)
	}

	// Update the release from the discogs pull
	record.GetRelease().ReleaseDate = release.GetReleaseDate()
	return b.db.SaveRecord(ctx, d.GetUserId(), record)
}
