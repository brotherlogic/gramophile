package background

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (b *BackgroundRunner) GetCollectionPage(ctx context.Context, page int32) error {
	releases, _, err := b.d.GetCollection(ctx, b.user.GetUser(), page)
	if err != nil {
		return err
	}

	for _, release := range releases {
		stored, err := b.db.GetRecord(ctx, b.user, release.GetInstanceId())

		if err == nil {
			if !proto.Equal(stored.GetRelease(), release) {
				stored.Release = release
				err = b.db.SaveRecord(ctx, b.user, stored)
				if err != nil {
					return err
				}
			}
		} else if status.Code(err) == codes.DataLoss {
			record := &pb.Record{Release: release}
			err = b.db.SaveRecord(ctx, b.user, record)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}

	return nil
}
