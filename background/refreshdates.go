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
		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Auth:    token,
				Entry: &pb.QueueElement_RefreshEarliestReleaseDate{
					RefreshEarliestReleaseDate: &pb.RefreshEarliestReleaseDate{
						Iid:                   iid,
						OtherRelease:          m.GetId(),
						UpdateDigitalWantlist: digWants,
					}}},
		})
		log.Printf("ENQUEED %v", iid)
		if err != nil {
			return fmt.Errorf("unable to queue sales: %v", err)
		}
	}

	return nil
}

func (b *BackgroundRunner) RefreshReleaseDate(ctx context.Context, d discogs.Discogs, digWants bool, iid, rid int64) error {
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
		qlog(ctx, "Updating ERD: %v", release)
		storedRelease.EarliestReleaseDate = release.GetReleaseDate()
		return b.db.SaveRecord(ctx, d.GetUserId(), storedRelease, &db.SaveOptions{})
	}

	addDigital := b.addDigitalList(ctx, storedRelease, release)
	needsSave = needsSave || addDigital

	log.Printf("DIG %v and %v", addDigital, digWants)
	if addDigital && digWants {
		updated := false
		// Update any wantlist if needed
		wantlist, err := b.db.LoadWantlist(ctx, d.GetUserId(), "digital_wantlist")
		if err != nil {
			return err
		}

		for _, dig := range storedRelease.GetDigitalIds() {
			found := false
			for _, ex := range wantlist.GetEntries() {
				if ex.GetId() == dig {
					found = true
				}
			}

			if !found {
				wantlist.Entries = append(wantlist.Entries, &pb.WantlistEntry{Id: dig})
				updated = true
			}
		}

		if updated {
			b.db.SaveWantlist(ctx, d.GetUserId(), wantlist)
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

	qlog(ctx, "DIGITAL %v is %v", childRelease, isDigital)

	if isDigital {
		storedRelease.DigitalIds = append(storedRelease.DigitalIds, childRelease.GetId())
	}

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

	return isDigital
}
