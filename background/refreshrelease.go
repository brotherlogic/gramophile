package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

type refreshReleaseHandler struct {
	b *BackgroundRunner
}

func (h *refreshReleaseHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.ProcessRefreshRelease(ctx, u, d, entry, enqueue)
}

func (h *refreshReleaseHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	if entry.GetRefreshRelease().GetIntention() == "" {
		Intention.With(prometheus.Labels{"intention": "REJECT"}).Inc()
		return status.Errorf(codes.InvalidArgument, "You must specify an intention for this refresh: %T", entry.GetEntry())
	}
	Intention.With(prometheus.Labels{"intention": entry.GetRefreshRelease().GetIntention()}).Inc()

	// Check for a marker
	marker, err := db.GetRefreshMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
	if err != nil {
		if status.Code(err) != codes.NotFound {
			return fmt.Errorf("Unable to get refresh marker: %w", err)
		}
	} else if marker > 0 && time.Since(time.Unix(0, marker)) < time.Hour*24 && entry.GetRefreshRelease().GetIntention() != "Manual Update" {
		MarkerCount.Inc()
		return status.Errorf(codes.AlreadyExists, "Refresh is in the queue: %v", time.Since(time.Unix(0, marker)))
	}

	err = db.SetRefreshMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
	if err != nil {
		return fmt.Errorf("Unable to write refresh marker: %w", err)
	}
	return nil
}

func (h *refreshReleaseHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

const (
	// Refresh core stats every week
	refreshStatsFrequency = time.Hour * 24 * 7

	// Min refresh frequency
	minRefreshFreq = time.Hour * 24 * 7 * 4

	digitalWantlistName = "digital"
)

func (b *BackgroundRunner) ProcessRefreshRelease(ctx context.Context, u *pb.StoredUser, d discogs.Discogs, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	err := b.RefreshRelease(ctx, entry.GetRefreshRelease().GetIid(), u, d, entry.GetForce() || entry.GetRefreshRelease().GetIntention() == "Manual Update")
	qlog(ctx, "Refreshing %v for %v -> %v", entry.GetRefreshRelease().GetIid(), entry.GetRefreshRelease().GetIid(), err)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			err = EnqueueWithIgnore(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Auth:      entry.GetAuth(),
					RunDate:   time.Now().UnixNano(),
					Intention: fmt.Sprintf("Refreshing collection from release release %v", entry.GetRefreshRelease().GetIid()),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
					},
				},
			}, enqueue)
			if err != nil {
				return err
			}
		}
	}
	derr := b.db.DeleteRefreshMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
	if derr != nil {
		return err
	}
	return derr
}

func (b *BackgroundRunner) RefreshRelease(ctx context.Context, iid int64, u *pb.StoredUser, d discogs.Discogs, force bool) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	if !force && record.GetHighPrice().GetValue() > 0 && time.Since(time.Unix(0, record.GetLastUpdateTime())) < minRefreshFreq {
		qlog(ctx, "Not refreshing %v as %v", iid, time.Since(time.Unix(0, record.GetLastUpdateTime())))
		return nil
	}

	qlog(ctx, "Refreshing %v (%v)", iid, time.Since(time.Unix(0, record.GetLastUpdateTime())))

	//if time.Since(time.Unix(0, record.GetLastUpdateTime())) < RefreshReleasePeriod {
	//	return nil
	//}

	release, err := d.GetRelease(ctx, record.GetRelease().GetId())
	if err != nil {
		return fmt.Errorf("unable to get release %v from discogs: %w", record.GetRelease().GetId(), err)
	}
	qlog(ctx, "Read release: %v", release)

	if force || record.GetHighPrice().GetValue() == 0 || time.Since(time.Unix(0, record.GetLastStatRefresh())) > refreshStatsFrequency {
		// Update the median sale price
		stats, err := d.GetReleaseStats(ctx, release.GetId())
		if err != nil && status.Code(err) != codes.NotFound {
			return err
		}
		if status.Code(err) == codes.NotFound {
			// Default values are $100 high and median, $5 low price
			stats = &pbd.ReleaseStats{
				HighPrice:   10000,
				MedianPrice: 10000,
				LowPrice:    500,
			}
			qlog(ctx, "Default Stats for %v == %v (%v)", iid, stats, err)
			record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
			record.LowPrice = &pbd.Price{Currency: "USD", Value: stats.GetLowPrice()}
			record.HighPrice = &pbd.Price{Currency: "USD", Value: stats.GetHighPrice()}

		} else {
			qlog(ctx, "Stats for %v == %v (%v)", iid, stats, err)
			record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
			record.LowPrice = &pbd.Price{Currency: "USD", Value: stats.GetLowPrice()}
			record.HighPrice = &pbd.Price{Currency: "USD", Value: stats.GetHighPrice()}
		}

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

	err = b.refreshWantlists(ctx, u, d, record)

	err = b.db.SaveRecord(ctx, d.GetUserId(), record, &db.SaveOptions{})
	qlog(ctx, "Updated %v -> %v (%v)", release.GetInstanceId(), record, err)
	return err
}

func (b *BackgroundRunner) refreshWantlists(ctx context.Context, u *pb.StoredUser, d discogs.Discogs, record *pb.Record) error {
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
				return b.db.SaveWantlist(ctx, u, &pb.Wantlist{
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
				return b.db.SaveWantlist(ctx, u, wantlist)
			}
		}
	}

	return nil
}
