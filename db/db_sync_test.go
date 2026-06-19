package db

import (
	"context"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestLastItemSyncedTimeUpdated(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	tdb := NewTestDB(pstore)
	ctx := context.Background()
	userid := int32(123)

	// Create user
	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: userid},
		UserToken: "token",
		Auth: &pb.GramophileAuth{Token: "123"},
	}
	err := tdb.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("unable to save user: %v", err)
	}

	// Wait a moment
	time.Sleep(time.Millisecond * 10)

	// Save record
	err = tdb.SaveRecord(ctx, userid, &pb.Record{
		Release: &pbd.Release{InstanceId: 100},
	}, &SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	// Sleep to allow background goroutine to finish
	time.Sleep(time.Millisecond * 100)

	// Verify user sync time
	updatedUser, err := tdb.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}

	if updatedUser.LastItemSyncedTime == 0 {
		t.Errorf("LastItemSyncedTime was not updated, still 0")
	}
}
