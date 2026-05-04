package db

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestSnapshotOrdering(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	tdb := NewTestDB(pstore)

	tdb.SaveSnapshot(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, "madeup", &pb.OrganisationSnapshot{
		Date: 123,
		Hash: "abc",
	})
	tdb.SaveSnapshot(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, "madeup", &pb.OrganisationSnapshot{
		Date: 2345,
		Hash: "xyz",
	})

	latest, err := tdb.GetLatestSnapshot(context.Background(), 123, "madeup")
	if err != nil {
		t.Fatalf("Unable to get latest snapshot: %v", err)
	}

	if latest.GetHash() != "xyz" {
		t.Errorf("Wrong hash returned: %v", latest)
	}

}

func TestSaveRecord_PreserveDates(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	tdb := NewTestDB(pstore)
	ctx := context.Background()

	// Save initial record with dates
	err := tdb.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 100,
			Id:         10,
			DateAdded:  1234,
		},
	})
	if err != nil {
		t.Fatalf("Initial save failed: %v", err)
	}

	// Save record with zero dates
	err = tdb.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 100,
			Id:         10,
			DateAdded:  0,
		},
	})
	if err != nil {
		t.Fatalf("Second save failed: %v", err)
	}

	// Read back and verify dates are preserved
	ret, err := tdb.GetRecord(ctx, 123, 100)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if ret.GetRelease().GetDateAdded() != 1234 {
		t.Errorf("DateAdded was not preserved: %v, want 1234", ret.GetRelease().GetDateAdded())
	}
}
