package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
)

func (b *BackgroundRunner) RefreshRelease(ctx context.Context, iid int64, d discogs.Discogs) error {

	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	log.Printf("Refreshing %v (%v)", iid, time.Since(time.Unix(0, record.GetLastUpdateTime())))

	//if time.Since(time.Unix(0, record.GetLastUpdateTime())) < RefreshReleasePeriod {
	//	return nil
	//}

	release, err := d.GetRelease(ctx, record.GetRelease().GetId())
	if err != nil {
		return fmt.Errorf("unable to get release %v from discogs: %w", record.GetRelease().GetId(), err)
	}

	// Update the median sale price
	stats, err := d.GetReleaseStats(ctx, release.GetId())
	if err != nil {
		return err
	}
	log.Printf("Stats for %v == %v (%v)", iid, stats, err)
	record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
	record.LowPrice = &pbd.Price{Currency: "USD", Value: stats.GetLowPrice()}

	// Update the release from the discogs pull
	record.GetRelease().ReleaseDate = release.GetReleaseDate()
	if record.GetEarliestReleaseDate() == 0 {
		record.EarliestReleaseDate = release.GetReleaseDate()
	}
	record.LastUpdateTime = time.Now().UnixNano()

	return b.db.SaveRecord(ctx, d.GetUserId(), record)
}
