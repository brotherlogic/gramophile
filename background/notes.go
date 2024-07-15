package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) ProcessIntents(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, auth string) error {
	log.Printf("BOUNCE: %v", b.db)
	user, err := b.db.GetUser(ctx, auth)
	if err != nil {
		return err
	}

	fields, err := d.GetFields(ctx)
	if err != nil {
		return err
	}

	err = b.ProcessSetClean(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessSetWidth(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessSetWeight(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessGoalFolder(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessSleeve(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessArrived(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessKeep(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessScore(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	return b.ProcessListenDate(ctx, d, r, i, user, fields)
}

func (b *BackgroundRunner) ProcessSetClean(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetCleanTime() == 0 {
		return nil
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

	err := d.SetField(ctx, r.GetRelease(), cfield, time.Unix(0, i.GetCleanTime()).Format("2006-01-02"))
	if err != nil {
		return err
	}

	r.LastCleanTime = i.GetCleanTime()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessListenDate(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetListenTime() == 0 {
		return nil
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

	err := d.SetField(ctx, r.GetRelease(), cfield, time.Unix(0, i.GetListenTime()).Format("2006-01-02"))
	if err != nil {
		return err
	}

	r.LastListenTime = i.GetListenTime()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessScore(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetNewScore() == 0 {
		return nil
	}

	// We use negative set scores to reset the score remotely
	if i.GetNewScore() < 0 {
		i.NewScore = 0
	}

	// err := d.SetRating(ctx, r.GetRelease(), i.GetNewScore())
	var err error
	if err != nil {
		return err
	}

	r.GetRelease().Rating = i.GetNewScore()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessGoalFolder(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetGoalFolder() == "" {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.GOAL_FOLDER_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Goal Folder field (from %+v)", fields)
	}

	err := d.SetField(ctx, r.GetRelease(), cfield, i.GetGoalFolder())
	if err != nil {
		return err
	}

	r.GoalFolder = i.GetGoalFolder()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessSetWidth(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetWidth() == 0 {
		return nil
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

	err := d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%v", i.GetWidth()))
	if err != nil {
		return err
	}

	r.Width = i.GetWidth()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessSetWeight(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetWeight() == 0 {
		return nil
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

	err := d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%v", i.GetWeight()))
	if err != nil {
		return err
	}

	r.Weight = i.GetWeight()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessSleeve(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetSleeve() == "" {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.SLEEVE_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Sleeve field (from %+v)", fields)
	}

	err := d.SetField(ctx, r.GetRelease(), cfield, i.GetSleeve())
	if err != nil {
		return err
	}

	r.Sleeve = i.GetSleeve()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessArrived(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetArrived() == 0 {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.ARRIVED_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate arrived field (from %+v)", fields)
	}

	err := d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%v", time.Unix(0, i.GetArrived()).Format("2006-01-02")))
	if err != nil {
		return err
	}

	r.Arrived = i.GetArrived()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}

func (b *BackgroundRunner) ProcessKeep(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetKeep() == pb.KeepStatus_KEEP_UNKNOWN {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.KEEP_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate keep field (from %+v), looking for %v", fields, config.KEEP_FIELD)
	}

	newKeep := fmt.Sprintf("%v", i.GetKeep())
	if i.GetKeep() == pb.KeepStatus_RESET {
		newKeep = ""
	}
	err := d.SetField(ctx, r.GetRelease(), cfield, newKeep)
	if err != nil {
		return err
	}

	r.KeepStatus = i.GetKeep()
	if i.GetKeep() == pb.KeepStatus_RESET {
		r.KeepStatus = pb.KeepStatus_KEEP_UNKNOWN
	}
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r)
}
