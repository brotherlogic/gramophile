package background

import (
	"context"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (b *BackgroundRunner) ProcessCollectionPage(ctx context.Context, d discogs.Discogs, page int32) error {
	releases, _, err := d.GetCollection(ctx, page)
	if err != nil {
		return err
	}

	for _, release := range releases {
		stored, err := b.db.GetRecord(ctx, d.GetUserId(), release.GetInstanceId())

		if err == nil {
			if !proto.Equal(stored.GetRelease(), release) {
				stored.Release = release
				err = b.db.SaveRecord(ctx, d.GetUserId(), stored)
				if err != nil {
					return err
				}
			}
		} else if status.Code(err) == codes.DataLoss {
			record := &pb.Record{Release: release}
			err = b.db.SaveRecord(ctx, d.GetUserId(), record)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
