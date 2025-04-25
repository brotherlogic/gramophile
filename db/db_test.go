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
