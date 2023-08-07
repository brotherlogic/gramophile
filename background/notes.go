package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) ProcessIntents(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, auth string) error {
	user, err := b.db.GetUser(ctx, auth)
	if err != nil {
		return err
	}

	err = b.ProcessSetClean(ctx, d, r, i, user)
	if err != nil {
		return err
	}

	err = b.ProcessSetWidth(ctx, d, r, i, user)
	if err != nil {
		return err
	}

	err = b.ProcessSetWeight(ctx, d, r, i, user)
	if err != nil {
		return err
	}

	return b.ProcessListenDate(ctx, d, r, i, user)
}

func (b *BackgroundRunner) ProcessSetClean(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser) error {
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
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessListenDate(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser) error {
	// We don't zero out the listen time
	if i.GetListenTime() == 0 {
		return nil
	}

	log.Printf("Getting fields: %v", d.GetUserId())
	fields, err := d.GetFields(ctx)
	if err != nil {
		return err
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.LISTEN_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Listen field (from %+v)", fields)
	}

	err = d.SetField(ctx, r.GetRelease(), cfield, time.Unix(i.GetListenTime(), 0).Format("2006-01-02"))
	if err != nil {
		return err
	}

	r.LastListenTime = i.GetListenTime()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessSetWidth(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser) error {
	// We don't zero out the clean time
	if i.GetWidth() == 0 {
		return nil
	}

	log.Printf("Getting fields: %v", d.GetUserId())
	fields, err := d.GetFields(ctx)
	if err != nil {
		return err
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.WIDTH_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Width field (from %+v)", fields)
	}

	err = d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%v", i.GetWidth()))
	if err != nil {
		return err
	}

	r.Width = i.GetWidth()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessSetWeight(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser) error {
	// We don't zero out the clean time
	if i.GetWeight() == 0 {
		return nil
	}

	log.Printf("Getting fields: %v", d.GetUserId())
	fields, err := d.GetFields(ctx)
	if err != nil {
		return err
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.WEIGHT_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate weight field (from %+v)", fields)
	}

	err = d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%v", i.GetWeight()))
	if err != nil {
		return err
	}

	r.Weight = i.GetWeight()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}
