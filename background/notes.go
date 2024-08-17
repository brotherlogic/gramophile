package background

import (
	"context"
	"fmt"
	"log"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	"github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"

	ghbclient "github.com/brotherlogic/githubridge/client"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/config"
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

	err = b.ProcessSetFolder(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	return b.ProcessListenDate(ctx, d, r, i, user, fields)
}

func (b *BackgroundRunner) buildRecord(ctx context.Context, userid int32, iid int64) (string, error) {
	rec, err := b.db.GetRecord(ctx, userid, iid)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v - %v", rec.GetRelease().GetArtists()[0].GetName(), rec.GetRelease().GetTitle()), nil
}

func (b *BackgroundRunner) buildLocation(ctx context.Context, org *pb.Organisation, s *pb.OrganisationSnapshot, index int32, nc int32, userid int32) (*pb.Location, error) {
	var before []*pb.Context
	var after []*pb.Context

	for i := index - 1; i >= max(0, index-nc); i-- {
		rec, err := b.buildRecord(ctx, userid, s.GetPlacements()[i].GetIid())
		if err != nil {
			return nil, err
		}

		before = append(before, &pb.Context{
			Index:  i,
			Iid:    s.GetPlacements()[i].GetIid(),
			Record: rec,
		})
	}

	for i := index + 1; i <= min(int32(len(s.GetPlacements())-1), index+nc); i++ {
		rec, err := b.buildRecord(ctx, userid, s.GetPlacements()[i].GetIid())
		if err != nil {
			return nil, err
		}

		after = append(after, &pb.Context{
			Index:  i,
			Iid:    s.GetPlacements()[i].GetIid(),
			Record: rec,
		})
	}

	return &pb.Location{
		LocationName: org.GetName(),
		Before:       before,
		After:        after,
		Slot:         s.GetPlacements()[index].GetUnit(),
	}, nil
}

func getOrg(folderId int32, config *pb.GramophileConfig) *pb.Organisation {
	for _, org := range config.GetOrganisationConfig().GetOrganisations() {
		for _, folder := range org.GetFoldersets() {
			if folder.GetFolder() == folderId {
				return org
			}
		}
	}
	return nil
}

func (b *BackgroundRunner) getLocation(ctx context.Context, userId int32, r *pb.Record, config *pb.GramophileConfig) (*pb.Location, error) {
	for _, org := range config.GetOrganisationConfig().GetOrganisations() {
		found := false
		for _, folder := range org.GetFoldersets() {
			if folder.GetFolder() == r.GetRelease().GetFolderId() {
				found = true
			}
		}

		if found {
			snapshot, err := b.db.GetLatestSnapshot(ctx, userId, org.GetName())
			if err != nil {
				return nil, err
			}

			index := -1
			for i, val := range snapshot.GetPlacements() {
				if val.GetIid() == r.GetRelease().GetInstanceId() {
					index = i
					break
				}
			}

			if index < 0 {
				return nil, status.Errorf(codes.Internal, "Record %v is listed to be in %v but does not appear in latest snapshot (%v -> %v)", r.GetRelease().GetInstanceId(), org.GetName(), snapshot.GetHash(), time.Unix(0, snapshot.GetDate()))
			}

			return b.buildLocation(ctx, org, snapshot, int32(index), config.GetPrintMoveConfig().GetContext(), userId)
		}
	}

	gclient, err := ghbclient.GetClientInternal()
	if err != nil {
		return nil, err
	}
	_, err = gclient.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
		User:  "brotherlogic",
		Repo:  "gramophile",
		Body:  fmt.Sprintf("Add %v to the org list", r.GetRelease().GetFolderId()),
		Title: "Add organisation",
	})
	log.Printf("Created issue -> %v", err)

	return nil, status.Errorf(codes.FailedPrecondition, "Unable to locate %v in an org (%v)", r.GetRelease().GetInstanceId(), r.GetRelease().GetFolderId())
}

func (b *BackgroundRunner) ProcessSetFolder(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the clean time
	if i.GetNewFolder() == 0 {
		return nil
	}

	// Quick exit if we're moving to where it already is
	if r.GetRelease().GetFolderId() == i.GetNewFolder() {
		return nil
	}

	// Move the record
	err := d.SetFolder(ctx,
		r.GetRelease().GetInstanceId(),
		r.GetRelease().GetId(),
		r.GetRelease().GetFolderId(), i.GetNewFolder())
	if err != nil {
		return err
	}

	// Run a preorg since this might be a new record
	orglogic := org.GetOrg(b.db)
	norg := getOrg(i.GetNewFolder(), user.GetConfig())
	if norg == nil {
		return status.Errorf(codes.Internal, "Unable to locate organisation for %v", i.GetNewFolder())
	}
	snap, err := orglogic.BuildSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()), user.GetConfig().GetOrganisationConfig())
	if err != nil {
		return err
	}
	log.Printf("Saving new snaphot: %v -> %v", snap.GetName(), snap.GetHash())
	b.db.SaveSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()).GetName(), snap)

	oldLoc, err := b.getLocation(ctx, user.GetUser().GetDiscogsUserId(), r, user.GetConfig())
	if err != nil {
		return fmt.Errorf("Unable to get prior location: %w", err)
	}

	r.GetRelease().FolderId = i.GetNewFolder()
	b.db.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), r)
	norg = getOrg(i.GetNewFolder(), user.GetConfig())
	if norg == nil {
		return status.Errorf(codes.Internal, "Unable to locate organisation for %v", i.GetNewFolder())
	}
	snap, err = orglogic.BuildSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()), user.GetConfig().GetOrganisationConfig())
	if err != nil {
		return err
	}
	log.Printf("Saving new snaphot: %v -> %v", snap.GetName(), snap.GetHash())
	b.db.SaveSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()).GetName(), snap)

	newLoc, err := b.getLocation(ctx, user.GetUser().GetDiscogsUserId(), r, user.GetConfig())
	if err != nil {
		return fmt.Errorf("Unable to get subsequent location: %w", err)
	}

	// Save the change for printing
	err = b.db.SavePrintMove(ctx, user.GetUser().GetDiscogsUserId(), &pb.PrintMove{
		Timestamp:   time.Now().UnixNano(),
		Iid:         r.GetRelease().GetInstanceId(),
		Origin:      oldLoc,
		Destination: newLoc,
		Record:      fmt.Sprintf("%v - %v", r.GetRelease().GetArtists()[0].GetName(), r.GetRelease().GetTitle()),
	})
	log.Printf("Savedthe print move for %v -> %v", r.GetRelease().GetInstanceId(), err)
	if err != nil {
		return err
	}

	// Clear the score if we've moved into the listening pile
	if i.GetNewFolder() == 812802 {
		qlog(ctx, "Setting rating for %v to zero", r.GetRelease().GetInstanceId())
		d.SetRating(ctx, r.GetRelease().GetId(), 0)
		r.GetRelease().Rating = 0
	}

	return b.db.SaveRecord(ctx, d.GetUserId(), r)
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

	qlog(ctx, "New score for %v (%v)", r.GetRelease().GetId(), i.GetNewScore())
	err := d.SetRating(ctx, r.GetRelease().GetId(), i.GetNewScore())
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
