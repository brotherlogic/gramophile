package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

const (
	// Refresh release dates every 6 months
	refreshRelaseDateFrequency = time.Hour * 24 * 7 * 4 * 6
)

type refreshEarliestReleaseDatesHandler struct {
	b *BackgroundRunner
}

func (h *refreshEarliestReleaseDatesHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	digWants := u.GetConfig().GetWantsConfig().GetDigitalWantList()
	err := h.b.RefreshReleaseDates(ctx, d, entry.GetAuth(), entry.GetRefreshEarliestReleaseDates().GetIid(), entry.GetRefreshEarliestReleaseDates().GetMasterId(), digWants, enqueue)
	if err != nil {
		return err
	}
	return h.b.db.DeleteRefreshDateMarker(ctx, entry.GetAuth())
}

func (h *refreshEarliestReleaseDatesHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *refreshEarliestReleaseDatesHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

type refreshEarliestReleaseDateHandler struct {
	b *BackgroundRunner
}

func (h *refreshEarliestReleaseDateHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.RefreshReleaseDate(ctx, u, d, entry.GetRefreshEarliestReleaseDate().GetUpdateDigitalWantlist(), entry.GetRefreshEarliestReleaseDate().GetIid(), entry.GetRefreshEarliestReleaseDate().GetOtherRelease(), entry.GetAuth(), enqueue)
}

func (h *refreshEarliestReleaseDateHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *refreshEarliestReleaseDateHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) RefreshReleaseDates(ctx context.Context, d discogs.Discogs, token string, iid, mid int64, digWants bool, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("Refreshing the MID %v", mid)

	// Don't refresh if record has no masters
	if mid == 0 {
		return nil
	}

	masters, err := d.GetMasterReleases(ctx, mid, 1, pbd.MasterSort_BY_YEAR)
	if err != nil {
		log.Printf("Can't get masters: %v", err)
		return err
	}

	log.Printf("FOUND MASTERS: %v -> %v", iid, masters)
	for _, m := range masters {
		err = EnqueueWithIgnore(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: "From refresh release dates",
				RunDate:   time.Now().UnixNano(),
				Auth:      token,
				Entry: &pb.QueueElement_RefreshEarliestReleaseDate{
					RefreshEarliestReleaseDate: &pb.RefreshEarliestReleaseDate{
						Iid:                   iid,
						OtherRelease:          m.GetId(),
						UpdateDigitalWantlist: digWants,
					}}},
		}, enqueue)
		log.Printf("ENQUEED %v", iid)
		if err != nil {
			return fmt.Errorf("unable to queue: %v", err)
		}
	}

	return nil
}

func (b *BackgroundRunner) RefreshReleaseDate(ctx context.Context, u *pb.StoredUser, d discogs.Discogs, digWants bool, iid, rid int64, token string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	log.Printf("STORED %v -> %v", iid, rid)
	storedRelease, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		log.Printf("Error in get record: %v", err)
		return err
	}

	release, err := d.GetRelease(ctx, rid)
	log.Printf("Release: %v", err)
	if err != nil {
		// We should be able to find any release here
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	needsSave := time.Since(time.Unix(0, storedRelease.GetLastEarliestReleaseUpdate())) > time.Hour
	storedRelease.LastEarliestReleaseUpdate = time.Now().UnixNano()

	log.Printf("GOT %v vs %v", release, storedRelease)
	if release.GetReleaseDate() < storedRelease.GetEarliestReleaseDate() || (release.GetReleaseDate() > 0 && storedRelease.GetEarliestReleaseDate() == 0) {
		storedRelease.EarliestReleaseDate = release.GetReleaseDate()
		qlog(ctx, "Updating ERD: %v -> %v", release, storedRelease.EarliestReleaseDate)

		err = b.db.SaveRecord(ctx, d.GetUserId(), storedRelease, &db.SaveOptions{})
		if err != nil {
			return err
		}
	}

	addDigital := b.addDigitalList(ctx, storedRelease, release)
	needsSave = needsSave || addDigital

	log.Printf("DIG %v and %v -> %v", addDigital, digWants, storedRelease)
	if addDigital && digWants && storedRelease.GetKeepStatus() == pb.KeepStatus_DIGITAL_KEEP {
		updated := false
		// Update any wantlist if needed
		wantlist, err := b.db.LoadWantlist(ctx, d.GetUserId(), "digital_wantlist")
		if err != nil {
			return err
		}
		log.Printf("ADDING: %v", wantlist)

		for _, dig := range storedRelease.GetDigitalIds() {
			found := false
			for _, ex := range wantlist.GetEntries() {
				if ex.GetId() == dig {
					found = true
				}
			}

			if !found {
				log.Printf("ADDING %v -> %v", dig, storedRelease.GetRelease().GetId())
				wantlist.Entries = append(wantlist.Entries, &pb.WantlistEntry{Id: dig, State: pb.WantState_WANTED, SourceId: iid})
				updated = true
			}
		}

		if updated {
			b.db.SaveWantlist(ctx, u, wantlist)

			// Since we updated the wants, we should also trigger a wants sync
			err = EnqueueWithIgnore(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention: "From refresh release date",
					RunDate:   time.Now().UnixNano(),
					Auth:      token,
					Entry:     &pb.QueueElement_RefreshWantlists{},
				},
			}, enqueue)
			if err != nil {
				return err
			}
		}
	}

	if needsSave {
		return b.db.SaveRecord(ctx, d.GetUserId(), storedRelease, &db.SaveOptions{})
	}

	return nil
}

func (b *BackgroundRunner) addDigitalList(ctx context.Context, storedRelease *pb.Record, childRelease *pbd.Release) bool {
	qlog(ctx, "Adding to digital for %v (%v)", storedRelease.GetRelease().GetInstanceId(), childRelease.GetId())
	// Is this release already in the list?
	for _, dig := range storedRelease.GetDigitalIds() {
		if dig == childRelease.GetId() {
			return false
		}
	}

	// Is this a digital release
	isDigital := false
	for _, format := range childRelease.GetFormats() {
		if format.GetName() == "CD" || format.GetName() == "CDr" || format.GetName() == "File" {
			isDigital = true
		}

		for _, desc := range format.GetDescriptions() {
			if desc == "CD" || desc == "CDr" || desc == "File" {
				isDigital = true
			}
		}
	}

	qlog(ctx, "DIGITAL %v is %v (%v)", childRelease, isDigital, storedRelease.DigitalIds)

	if isDigital {
		storedRelease.DigitalIds = append(storedRelease.DigitalIds, childRelease.GetId())

		found := false
		for _, entry := range storedRelease.GetDigitalVersions() {
			if entry.GetId() == childRelease.GetId() {
				found = true
			}
		}
		if !found {
			storedRelease.DigitalVersions = append(storedRelease.DigitalVersions, &pb.DigitalVersion{
				Id:                   childRelease.GetId(),
				DigitalVersionSource: pb.DigitalVersion_DIGITAL_VERSION_SOURCE_COMPUTED,
			})
		}
	}
	qlog(ctx, "BUTNOW %v", storedRelease.DigitalVersions)

	return isDigital
}
