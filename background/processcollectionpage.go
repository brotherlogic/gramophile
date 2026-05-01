package background

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/config"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (b *BackgroundRunner) processNotes(ctx context.Context, field []*pbd.Field, r *pb.Record) (*pb.Record, error) {
	for key, value := range r.GetRelease().GetNotes() {
		for _, f := range field {
			if f.GetId() == key {
				switch f.GetName() {
				case config.CLEANED_FIELD_NAME:
					val, err := time.Parse("2006-01-02", value)
					if err != nil {
						return nil, fmt.Errorf("unable to parse %v as date: %w", value, err)
					}
					r.LastCleanTime = val.UnixNano()
				case config.ARRIVED_FIELD:
					val, err := time.Parse("2006-01-02", value)
					if err != nil {
						return nil, err
					}
					r.Arrived = val.UnixNano()
				case config.WIDTH_FIELD:
					val, err := strconv.ParseFloat(value, 32)
					if err != nil {
						return nil, err
					}
					r.Width = float32(val)
				case config.SLEEVE_FIELD:
					r.Sleeve = value
				}
			}
		}
	}

	// Clear the remaining notes
	r.GetRelease().Notes = make(map[int32]string)

	return r, nil
}

var (
	crefresh = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_refreshid",
		Help: "The length of the working queue I think yes",
	})
)

func (b *BackgroundRunner) ProcessRefreshCollectionEntry(ctx context.Context, d discogs.Discogs, user *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	if entry.GetRefreshCollectionEntry().GetPage() == 1 {
		entry.GetRefreshCollectionEntry().RefreshId = time.Now().UnixNano()
	}

	rval, err := b.ProcessCollectionPage(ctx, d, entry.GetRefreshCollectionEntry().GetPage(), entry.GetRefreshCollectionEntry().GetRefreshId())
	qlog(ctx, "Processed collection page: %v %v", rval, err)

	if err != nil {
		return err
	}

	isSetIntentMiss := strings.HasPrefix(entry.GetIntention(), "Triggered from miss on SetIntent")
	if isSetIntentMiss && rval > 2 {
		rval = 2
	}

	if entry.GetRefreshCollectionEntry().GetPage() == 1 {
		for i := int32(2); i <= rval; i++ {
			_, err = enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
				Force:     entry.GetForce(),
				RunDate:   time.Now().UnixNano() + int64(i),
				Intention: entry.GetIntention(),
				Entry: &pb.QueueElement_RefreshCollectionEntry{
					RefreshCollectionEntry: &pb.RefreshCollectionEntry{
						Page: i, RefreshId: entry.GetRefreshCollectionEntry().GetRefreshId()}},
				Auth: entry.GetAuth(),
			}})
			if err != nil {
				return fmt.Errorf("unable to enqueue: %w", err)
			}
		}
		user.LastCollectionRefresh = time.Now().UnixNano()
		err = b.db.SaveUser(ctx, user)
		if err != nil {
			return fmt.Errorf("unable to save user: %w", err)
		}
	}

	if entry.GetRefreshCollectionEntry().GetPage() == rval {
		qlog(ctx, "Writing collection refresh chip")
		//Move records
		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: entry.GetIntention(),
				Force:     entry.GetForce(),
				RunDate:   time.Now().UnixNano() + int64(rval) + 10,
				Entry: &pb.QueueElement_MoveRecords{
					MoveRecords: &pb.MoveRecords{}},
				Auth: entry.GetAuth(),
			}})
		qlog(ctx, "Found %v", err)
		if user.GetState() == pb.StoredUser_USER_STATE_REFRESHING {
			user.State = pb.StoredUser_USER_STATE_IN_WAITLIST
			err = b.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to save user: %w", err)
			}
		}
		if err != nil {
			return err
		}
		if !isSetIntentMiss {
			return b.CleanCollection(ctx, d, entry.GetRefreshCollectionEntry().GetRefreshId(), entry.GetAuth(), enqueue)
		}
		return nil
	}
	return nil
}

func (b *BackgroundRunner) ProcessCollectionPage(ctx context.Context, d discogs.Discogs, page int32, refreshId int64) (int32, error) {
	crefresh.Set(float64(refreshId))

	releases, pag, err := d.GetCollection(ctx, page)
	if err != nil {
		return -1, err
	}
	log.Printf("Found %v releases in collection", len(releases))

	fields, err := d.GetFields(ctx)
	if err != nil {
		return -1, fmt.Errorf("unable to get fields: %w", err)
	}

	for _, release := range releases {
		if err != nil {
			return -1, fmt.Errorf("unable to get release stats for %v: %w", release.GetId(), err)
		}

		stored, err := b.db.GetRecord(ctx, d.GetUserId(), release.GetInstanceId())

		if err == nil && stored != nil {
			log.Printf("Huh: %v and %v", stored, release)

			stored.Release.Artists = []*pbd.Artist{}
			stored.Release.Formats = []*pbd.Format{}
			stored.Release.Labels = []*pbd.Label{}
			proto.Merge(stored.Release, release)
			stored.RefreshId = refreshId

			// Process the notes
			stored, err = b.processNotes(ctx, fields, stored)
			if err != nil {
				return -1, err
			}

			err = b.db.SaveRecord(ctx, d.GetUserId(), stored, &db.SaveOptions{})
			if err != nil {
				return -1, err
			}
		} else if status.Code(err) == codes.NotFound {
			record := &pb.Record{Release: release}
			record.RefreshId = refreshId
			//record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}

			// Process the notes
			record, err = b.processNotes(ctx, fields, record)
			if err != nil {
				return -1, err
			}

			err = b.db.SaveRecord(ctx, d.GetUserId(), record, &db.SaveOptions{})
			if err != nil {
				return -1, err
			}
		} else {
			return -1, err
		}
	}

	return pag.GetPages(), nil
}
