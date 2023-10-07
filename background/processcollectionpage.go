package background

import (
	"context"
	"fmt"
	"log"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) ProcessCollectionPage(ctx context.Context, d discogs.Discogs, page int32, refreshId int64) (int32, error) {
	releases, pag, err := d.GetCollection(ctx, page)
	if err != nil {
		return -1, err
	}

	for _, release := range releases {
		stats, err := d.GetReleaseStats(ctx, release.GetId())
		if err != nil {
			return -1, fmt.Errorf("unable to get release stats: %w", err)
		}

		stored, err := b.db.GetRecord(ctx, d.GetUserId(), release.GetInstanceId())

		if err == nil && stored != nil {
			log.Printf("Huh: %v and %v", stored, release)
			stored.Release = release
			stored.RefreshId = refreshId

			stored.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
			stored.LastUpdateTime = time.Now().Unix()

			err = b.db.SaveRecord(ctx, d.GetUserId(), stored)
			if err != nil {
				return -1, err
			}
		} else if status.Code(err) == codes.NotFound {
			record := &pb.Record{Release: release}
			record.RefreshId = refreshId
			record.MedianPrice = &pbd.Price{Currency: "USD", Value: stats.GetMedianPrice()}
			record.LastUpdateTime = time.Now().Unix()

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
