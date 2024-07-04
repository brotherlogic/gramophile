package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

const (
	// Refresh core stats every week
	refreshStatsFrequency = time.Hour * 24 * 7

	// Min refresh frequency
	minRefreshFreq = time.Hour * 24 * 7

	digitalWantlistName = "digital"
)

func (b *BackgroundRunner) RefreshRelease(ctx context.Context, iid int64, d discogs.Discogs, force bool) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	if !force && time.Since(time.Unix(0, record.GetLastUpdateTime())) < minRefreshFreq {
		log.Printf("Not refreshing %v as %v", iid, time.Since(time.Unix(0, record.GetLastUpdateTime())))
		return nil
	}

	log.Printf("Refreshing %v (%v)", iid, time.Since(time.Unix(0, record.GetLastUpdateTime())))

	//if time.Since(time.Unix(0, record.GetLastUpdateTime())) < RefreshReleasePeriod {
	//	return nil
	//}

	release, err := d.GetRelease(ctx, record.GetRelease().GetId())
	if err != nil {
		return fmt.Errorf("unable to get release %v from discogs: %w", record.GetRelease().GetId(), err)
	}

	log.Printf("Checking stats")
	if time.Since(time.Unix(0, record.GetLastStatRefresh())) > refreshStatsFrequency {
		// Update the median sale price
		stats, err := d.GetReleaseStats(ctx, release.GetId())
		if err != nil {
			return err
		}
		log.Printf("Stats for %v == %v (%v)", iid, stats, err)
		record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
		record.LowPrice = &pbd.Price{Currency: "USD", Value: stats.GetLowPrice()}

		record.LastStatRefresh = time.Now().UnixNano()
	} else {
		log.Printf("Skipping stats because %v", time.Unix(0, record.GetLastStatRefresh()))
	}

	// Update the release from the discogs pull
	record.GetRelease().ReleaseDate = release.GetReleaseDate()
	if record.GetEarliestReleaseDate() == 0 {
		record.EarliestReleaseDate = release.GetReleaseDate()
	}

	// Clear repeated fields so they don't get concatenated when we merge
	record.GetRelease().Artists = []*pbd.Artist{}
	record.GetRelease().Formats = []*pbd.Format{}
	record.GetRelease().Labels = []*pbd.Label{}
	proto.Merge(record.Release, release)

	record.LastUpdateTime = time.Now().UnixNano()

	//TODO: Need to pull the instance specifc details in here

	err = b.refreshWantlists(ctx, d, record)

	err = b.db.SaveRecord(ctx, d.GetUserId(), record)
	log.Printf("Updated %v -> %v (%v)", release.GetInstanceId(), record, err)
	return err
}

func (b *BackgroundRunner) refreshWantlists(ctx context.Context, d discogs.Discogs, record *pb.Record) error {
	if record.GetKeepStatus() == pb.KeepStatus_DIGITAL_KEEP {
		isPhysical := false
		for _, format := range record.GetRelease().GetFormats() {
			if format.GetName() == "Vinyl" {
				isPhysical = true
			}
		}

		if isPhysical {
			wantlist, err := b.db.LoadWantlist(ctx, d.GetUserId(), digitalWantlistName)
			if err != nil && status.Code(err) == codes.NotFound {
				// Create a digital wantlist here
				return b.db.SaveWantlist(ctx, d.GetUserId(), &pb.Wantlist{
					Name:    digitalWantlistName,
					Type:    pb.WantlistType_ONE_BY_ONE,
					Entries: []*pb.WantlistEntry{{Id: record.GetRelease().GetId()}},
				})
			}
			if err != nil {
				return err
			}

			found := false
			for _, entry := range wantlist.GetEntries() {
				if entry.GetId() == record.GetRelease().GetId() {
					found = true
				}
			}
			if !found {
				wantlist.Entries = append(wantlist.Entries, &pb.WantlistEntry{MasterId: record.GetRelease().GetMasterId(), DigitalOnly: true})
				return b.db.SaveWantlist(ctx, d.GetUserId(), wantlist)
			}
		}
	}

	return nil
}
