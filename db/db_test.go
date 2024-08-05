package db

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestUpdateDiff(t *testing.T) {
	tests := []struct {
		update  *pb.RecordUpdate
		diffstr string
	}{
		{
			update: &pb.RecordUpdate{
				Before: &pb.Record{
					GoalFolder: "",
				},
				After: &pb.Record{
					GoalFolder: "12 Inches",
				},
			},
			diffstr: "Goal Folder was set to 12 Inches",
		},
	}

	for _, tc := range tests {
		rd := ResolveDiff(tc.update)
		found := false
		for _, rdd := range rd {
			if rdd == tc.diffstr {
				found = true
			}
		}
		if !found {
			t.Errorf("bad diff: expected %v, got %v", tc.diffstr, rd)
		}

	}
}

func TestSnapshotOrdering(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	tdb := NewTestDB(rstore)

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
