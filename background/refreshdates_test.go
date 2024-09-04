package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	dpb "github.com/brotherlogic/discogs/proto"
	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestDigitalListExtended(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	d := &discogs.TestDiscogsClient{UserId: 123}
	b.db.SaveRecord(context.Background(), 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 100, MasterId: 200,
		}})

	d.AddCollectionRelease(&dpb.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})

	d.AddCNonollectionRelease(&dpb.Release{MasterId: 200, Id: 2, Rating: 2})
	d.AddCNonollectionRelease(&dpb.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})

	b.RefreshReleaseDate(context.Background(), d, false, 100, 2)
	b.RefreshReleaseDate(context.Background(), d, false, 100, 3)

	rec, err := b.db.GetRecord(context.Background(), 123, 100)
	if err != nil {
		t.Errorf("Bad release pull: %v", err)
	}
	if len(rec.GetDigitalIds()) != 1 || rec.GetDigitalIds()[0] != 3 {
		t.Errorf("Bad digital ids: %v", rec.GetDigitalIds())
	}
}
