package background

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) ProcessIntents(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, auth string) error {
	err := b.ProcessSetClean(ctx, d, r, i)
	if err != nil {
		return err
	}

	user, err := b.db.GetUser(ctx, auth)
	if err != nil {
		return err
	}

	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), r)
}

func (b *BackgroundRunner) ProcessSetClean(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent) error {
	// We don't zero out the clean time
	if i.GetCleanTime() == 0 {
		return nil
	}

	log.Printf("Getting fields: %v", d.GetUserId())
	fields, err := d.GetFields(ctx)
	if err != nil {
		return err
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == "Cleaned" {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Cleaned field (from %+v)", fields)
	}

	err = d.SetField(ctx, r.GetRelease(), cfield, time.Unix(i.GetCleanTime(), 0).Format("2006-01-02"))
	if err != nil {
		return err
	}

	r.LastCleanTime = i.GetCleanTime()
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}
