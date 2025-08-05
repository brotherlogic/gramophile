package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	dpb "github.com/brotherlogic/discogs/proto"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestDigitalListExtended(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	err = b.db.SaveWantlist(context.Background(), 123, &pb.Wantlist{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Bad wantlist save: %v", err)
	}

	d := &discogs.TestDiscogsClient{UserId: 123}
	b.db.SaveRecord(context.Background(), 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 100, MasterId: 200,
		}}, &db.SaveOptions{})

	d.AddCollectionRelease(&dpb.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})

	d.AddNonCollectionRelease(&dpb.Release{MasterId: 200, Id: 2, Rating: 2})
	d.AddNonCollectionRelease(&dpb.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})

	// Set the keep status
	rec, err := b.db.GetRecord(context.Background(), 123, 100)
	if err != nil {
		t.Errorf("Bad release pull: %v", err)
	}
	rec.KeepStatus = pb.KeepStatus_DIGITAL_KEEP
	err = b.db.SaveRecord(context.Background(), 123, rec)

	err = b.RefreshReleaseDate(context.Background(), d, true, 100, 2, "123", func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		//Do Nothing
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Bad refresh: %v", err)
	}
	b.RefreshReleaseDate(context.Background(), d, true, 100, 3, "123", func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		//Do Nothing
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Bad refresh: %v", err)
	}

	rec, err = b.db.GetRecord(context.Background(), 123, 100)
	if err != nil {
		t.Errorf("Bad release pull: %v", err)
	}
	rec.KeepStatus = pb.KeepStatus_DIGITAL_KEEP
	err = b.db.SaveRecord(context.Background(), 123, rec)
	if len(rec.GetDigitalIds()) != 1 || rec.GetDigitalIds()[0] != 3 {
		t.Errorf("Bad digital ids: %v", rec.GetDigitalIds())
	}

	wl, err := b.db.LoadWantlist(context.Background(), 123, "digital_wantlist")
	if err != nil {
		t.Fatalf("Bad wl: %v", err)
	}

	if len(wl.GetEntries()) != 1 {
		t.Errorf("Bad wantlist entries (should be 1): %v", wl)
	}
}
