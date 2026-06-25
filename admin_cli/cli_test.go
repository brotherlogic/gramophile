package main

import (
	"strings"
	"testing"

	discogs "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestFormatWaitlist(t *testing.T) {
	res := &pb.GetWaitlistStatusResponse{
		Users: []*pb.WaitlistUser{
			{
				User: &pb.StoredUser{
					User: &discogs.User{Username: "testuser1"},
					ExpectedCollectionSize: 100,
					ExpectedWantlistSize: 50,
				},
				SyncedCollectionSize: 50,
				SyncedWantlistSize: 25,
				IsStuck: false,
				EtaSeconds: 3600,
				FullySynced: false,
			},
			{
				User: &pb.StoredUser{
					User: &discogs.User{Username: "testuser2"},
					ExpectedCollectionSize: 10,
					ExpectedWantlistSize: 5,
				},
				SyncedCollectionSize: 10,
				SyncedWantlistSize: 5,
				IsStuck: false,
				EtaSeconds: 0,
				FullySynced: true,
			},
			{
				User: &pb.StoredUser{
					User: &discogs.User{Username: "testuser3"},
					ExpectedCollectionSize: 200,
					ExpectedWantlistSize: 100,
				},
				SyncedCollectionSize: 50,
				SyncedWantlistSize: 10,
				IsStuck: true,
				EtaSeconds: 10000,
				FullySynced: false,
			},
		},
	}

	output := formatWaitlist(res)
	if !strings.Contains(output, "testuser1") {
		t.Errorf("Output does not contain testuser1")
	}
	if !strings.Contains(output, "Partially Synced") {
		t.Errorf("Output does not contain Partially Synced")
	}
	if !strings.Contains(output, "Fully Synced") {
		t.Errorf("Output does not contain Fully Synced")
	}
	if !strings.Contains(output, "STUCK") {
		t.Errorf("Output does not contain STUCK")
	}
}
