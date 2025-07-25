package background

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"

	ghbclient "github.com/brotherlogic/githubridge/client"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/config"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) ProcessIntents(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, auth string, enqueue func(ctx context.Context, entry *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
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

	err = b.ProcessKeep(ctx, d, r, i, user, fields, auth, enqueue)
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

	err = b.ProcessSetOversize(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessPurchasePrice(ctx, d, r, i, user, fields)
	if err != nil {
		return err
	}

	err = b.ProcessPurchaseLocation(ctx, d, r, i, user, fields)
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

	if len(rec.GetRelease().GetArtists()) == 0 {
		return fmt.Sprintf("Unknown artist - %v", rec.GetRelease().GetTitle()), nil
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
		Shelf:        s.GetPlacements()[index].GetSpace(),
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
				return nil, status.Errorf(codes.Internal, "Record %v in %v is listed to be in %v but does not appear in latest snapshot (%v -> %v)", r.GetRelease().GetInstanceId(), r.GetRelease().GetFolderId(), org.GetName(), snapshot.GetHash(), time.Unix(0, snapshot.GetDate()))
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
	log.Printf("Setting folder %v -> %v", r.GetRelease(), i.GetNewFolder())

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
	var oldLoc *pb.Location
	if r.GetRelease().GetFolderId() == 0 {
		oldLoc = &pb.Location{
			LocationName: "New",
			Slot:         1,
			Shelf:        "New",
			Before:       []*pb.Context{},
			After:        []*pb.Context{},
		}
	} else {
		org := getOrg(r.GetRelease().GetFolderId(), user.GetConfig())
		if org == nil {
			return status.Errorf(codes.Internal, "Unable to locate old organisation for %v", r.GetRelease().GetFolderId())
		}
		snap, err := orglogic.BuildSnapshot(ctx, user, getOrg(r.GetRelease().GetFolderId(), user.GetConfig()), user.GetConfig().GetOrganisationConfig())
		if err != nil {
			return err
		}
		log.Printf("Saving new snaphot: %v -> %v", snap.GetName(), snap.GetHash())
		b.db.SaveSnapshot(ctx, user, getOrg(r.GetRelease().GetFolderId(), user.GetConfig()).GetName(), snap)

		oldLoc, err = b.getLocation(ctx, user.GetUser().GetDiscogsUserId(), r, user.GetConfig())
		if err != nil {
			return fmt.Errorf("Unable to get prior location (with %v @ %v): %w", snap.GetHash(), time.Unix(0, snap.GetDate()), err)
		}
	}

	r.GetRelease().FolderId = i.GetNewFolder()
	b.db.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), r, &db.SaveOptions{})
	norg := getOrg(i.GetNewFolder(), user.GetConfig())
	if norg == nil {
		return status.Errorf(codes.Internal, "Unable to locate new organisation for %v", i.GetNewFolder())
	}
	snap, err := orglogic.BuildSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()), user.GetConfig().GetOrganisationConfig())
	if err != nil {
		return err
	}
	log.Printf("Saving new snaphot: %v -> %v", snap.GetName(), snap.GetHash())
	b.db.SaveSnapshot(ctx, user, getOrg(i.GetNewFolder(), user.GetConfig()).GetName(), snap)

	newLoc, err := b.getLocation(ctx, user.GetUser().GetDiscogsUserId(), r, user.GetConfig())
	if err != nil {
		return fmt.Errorf("Unable to get subsequent location: %w", err)
	}

	artist := "UNKNOWN"
	if len(r.GetRelease().GetArtists()) > 0 {
		artist = r.GetRelease().GetArtists()[0].GetName()
	}

	// Save the change for printing
	err = b.db.SavePrintMove(ctx, user.GetUser().GetDiscogsUserId(), &pb.PrintMove{
		Timestamp:   time.Now().UnixNano(),
		Iid:         r.GetRelease().GetInstanceId(),
		Origin:      oldLoc,
		Destination: newLoc,
		Record:      fmt.Sprintf("%v - %v", artist, r.GetRelease().GetTitle()),
	})
	log.Printf("Savedthe print move for %v -> %v", r.GetRelease().GetInstanceId(), err)
	if err != nil {
		return err
	}

	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
}

func mapDiscogsScore(score int32, config *pb.ScoreConfig) int32 {
	if config.GetBottomRange() >= config.GetTopRange() {
		return score
	}
	rangeWidth := config.GetTopRange() - config.GetBottomRange()
	return int32(math.Ceil(5 * (float64(score) / float64(rangeWidth))))
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

	discogsScore := mapDiscogsScore(i.GetNewScore(), user.GetConfig().GetScoreConfig())

	qlog(ctx, "New score for %v (%v)", r.GetRelease().GetId(), i.GetNewScore())
	err := d.SetRating(ctx, r.GetRelease().GetId(), discogsScore)
	if err != nil {
		return err
	}

	r.GetRelease().Rating = discogsScore
	r.ScoreHistory = append(r.GetScoreHistory(), &pb.Score{
		ScoreValue:                i.GetNewScore(),
		ScoreMappedTo:             discogsScore,
		AppliedToDiscogsTimestamp: time.Now().UnixNano(),
	})

	config.Apply(user.GetConfig(), r)
	err = b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
	if err != nil {
		return err
	}

	if i.GetNewScore() > 0 {
		// Update a want with the given score
		want, err := b.db.GetWant(ctx, d.GetUserId(), r.GetRelease().GetId())
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return err
			}
			return nil
		}

		want.Score = i.GetNewScore()
		return b.db.SaveWant(ctx, d.GetUserId(), want, "Saving with new score")
	}

	return nil

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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
}

func (b *BackgroundRunner) ProcessPurchasePrice(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetPurchasePrice() == 0 {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.PURCHASED_PRICE_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Purchase Price field (from %+v)", fields)
	}

	err := d.SetField(ctx, r.GetRelease(), cfield, fmt.Sprintf("%.2f", float32(i.GetPurchasePrice())/100.0))
	if err != nil {
		return err
	}

	r.PurchasePrice = i.GetPurchasePrice()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
}

func (b *BackgroundRunner) ProcessPurchaseLocation(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	// We don't zero out the listen time
	if i.GetPurchaseLocation() == "" {
		return nil
	}

	cfield := -1
	for _, field := range fields {
		if field.GetName() == config.PURCHASED_LOCATION_FIELD {
			cfield = int(field.GetId())
		}
	}

	if cfield < 0 {
		return status.Errorf(codes.FailedPrecondition, "Unable to locate Purchase Location field (from %+v) -> %v", fields, config.PURCHASED_LOCATION_FIELD)
	}

	err := d.SetField(ctx, r.GetRelease(), cfield, i.GetPurchaseLocation())
	if err != nil {
		return err
	}

	r.PurchaseLocation = i.GetPurchaseLocation()
	config.Apply(user.GetConfig(), r)
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
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
	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
}

func (b *BackgroundRunner) ProcessSetOversize(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field) error {
	log.Printf("Processing Set Oversize")

	switch i.GetSetOversize() {
	case pb.Intent_SET:
		r.IsOversized = true
	case pb.Intent_UNSET:
		r.IsOversized = false
	}

	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})

}

func (b *BackgroundRunner) ProcessKeep(ctx context.Context, d discogs.Discogs, r *pb.Record, i *pb.Intent, user *pb.StoredUser, fields []*pbd.Field, auth string, enqueue func(ctx context.Context, entry *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("Processing Keep")

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

	// If this is an adjust to DIGITAL_KEEP - let's reset the refresh date in
	// order to refresh the digital wants
	if r.KeepStatus == pb.KeepStatus_DIGITAL_KEEP {
		r.LastEarliestReleaseUpdate = 0

		// Also enqueue an update for this relaese
		enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Auth:    auth,
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_RefreshEarliestReleaseDates{
					RefreshEarliestReleaseDates: &pb.RefreshEarliestReleaseDates{
						Iid:      r.GetRelease().GetInstanceId(),
						MasterId: r.GetRelease().GetMasterId(),
					}},
			},
		})

		for _, eid := range i.GetDigitalIds() {
			found := false
			for _, exid := range r.GetDigitalVersions() {
				if exid.GetId() == eid {
					exid.DigitalVersionSource = pb.DigitalVersion_DIGITAL_VERSION_SOURCE_PROVIDED
					found = true
				}
			}
			if !found {
				r.DigitalVersions = append(r.DigitalVersions, &pb.DigitalVersion{
					Id:                   eid,
					DigitalVersionSource: pb.DigitalVersion_DIGITAL_VERSION_SOURCE_PROVIDED,
				})
			}
		}
	}

	log.Printf("Trying to set: %v", r)
	if r.KeepStatus == pb.KeepStatus_MINT_UP_KEEP && user.GetConfig().GetWantsConfig().GetMintUpWantList() {
		for _, mid := range i.GetMintIds() {
			found := false
			for _, emid := range r.GetMintVersions() {
				if mid == emid {
					found = true
					break
				}
			}

			if !found {
				r.MintVersions = append(r.MintVersions, mid)
			}
		}

		// Now ensure that the mint up wantlist is up to date
		wl, err := b.db.LoadWantlist(ctx, user.GetUser().GetDiscogsUserId(), "mint_up_wantlist")
		if err != nil {
			return err
		}

		log.Printf("Working with %v", r.GetMintVersions())

		adjust := false
		for _, mid := range r.GetMintVersions() {
			found := false
			for _, entry := range wl.GetEntries() {
				if entry.GetId() == mid {
					found = true
					break
				}
			}

			if !found {
				wl.Entries = append(wl.Entries, &pb.WantlistEntry{
					Id: mid,
				})
				adjust = true
			}
		}

		if adjust {
			err = b.db.SaveWantlist(ctx, user.GetUser().GetDiscogsUserId(), wl)
			if err != nil {
				return err
			}
		}
	}

	return b.db.SaveRecord(ctx, d.GetUserId(), r, &db.SaveOptions{})
}
