package background

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
					r.LastCleanTime = val.Unix()
				case config.ARRIVED_FIELD:
					val, err := time.Parse("2006-01-02", value)
					if err != nil {
						return nil, err
					}
					r.LastCleanTime = val.Unix()
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

func (b *BackgroundRunner) ProcessCollectionPage(ctx context.Context, d discogs.Discogs, page int32, refreshId int64) (int32, error) {
	releases, pag, err := d.GetCollection(ctx, page)
	if err != nil {
		return -1, err
	}

	fields, err := d.GetFields(ctx)
	if err != nil {
		return -1, fmt.Errorf("unable to get fields: %w", err)
	}

	for _, release := range releases {
		/*stats, err := d.GetReleaseStats(ctx, release.GetId())
		if err != nil {
			return -1, fmt.Errorf("unable to get release stats for %v: %w", release.GetId(), err)
		}*/

		stored, err := b.db.GetRecord(ctx, d.GetUserId(), release.GetInstanceId())

		if err == nil && stored != nil {
			log.Printf("Huh: %v and %v", stored, release)
			stored.Release = release
			stored.RefreshId = refreshId

			//stored.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
			//stored.LastUpdateTime = time.Now().UnixNano()

			// Process the notes
			stored, err = b.processNotes(ctx, fields, stored)
			if err != nil {
				return -1, err
			}

			err = b.db.SaveRecord(ctx, d.GetUserId(), stored)
			if err != nil {
				return -1, err
			}
		} else if status.Code(err) == codes.NotFound {
			record := &pb.Record{Release: release}
			record.RefreshId = refreshId
			r //ecord.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}

			// Process the notes
			record, err = b.processNotes(ctx, fields, record)
			if err != nil {
				return -1, err
			}

			err = b.db.SaveRecord(ctx, d.GetUserId(), record)
			if err != nil {
				return -1, err
			}
		} else {
			return -1, err
		}
	}

	return pag.GetPages(), nil
}
