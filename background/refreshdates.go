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
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefreshReleaseDates(ctx context.Context, d discogs.Discogs, token string, iid, mid int64, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
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

	log.Printf("FOUND MASTERS: %v", masters)
	for _, m := range masters {
		_, err = enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Auth:    token,
				Entry: &pb.QueueElement_RefreshEarliestReleaseDate{
					RefreshEarliestReleaseDate: &pb.RefreshEarliestReleaseDate{
						Iid:          iid,
						OtherRelease: m.GetId(),
					}}},
		})
		log.Printf("ENQUEED %v", iid)
		if err != nil {
			return fmt.Errorf("unable to queue sales: %v", err)
		}
	}

	return nil
}

func (b *BackgroundRunner) RefreshReleaseDate(ctx context.Context, d discogs.Discogs, iid, rid int64) error {

	storedRelease, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return err
	}

	release, err := d.GetRelease(ctx, rid)
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
		//log.Printf("Updating ERD: %v", release)
		storedRelease.EarliestReleaseDate = release.GetReleaseDate()
		return b.db.SaveRecord(ctx, d.GetUserId(), storedRelease)
	}

	if needsSave {
		return b.db.SaveRecord(ctx, d.GetUserId(), storedRelease)
	}

	return nil
}
