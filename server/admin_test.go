package server

import (
	"fmt"
	"testing"
	"time"

	discogs "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestClean(t *testing.T) {
	ctx := getTestContext(123)
	d := db.NewTestDB(pstore_client.GetTestClient())
	s := Server{d: d}

	_, err := s.Clean(ctx, &pb.CleanRequest{})
	if err != nil {
		t.Errorf("Unable to run clean: %v", err)
	}
}

func TestWaitlistStatus(t *testing.T) {
	ctx := getTestContext(123)
	d := db.NewTestDB(pstore_client.GetTestClient())
	s := Server{d: d}

	// 0 users case
	res, err := s.GetWaitlistStatus(ctx, &pb.GetWaitlistStatusRequest{})
	if err != nil {
		t.Fatalf("GetWaitlistStatus returned error: %v", err)
	}
	if len(res.GetUsers()) != 0 {
		t.Errorf("Expected 0 users, got %v", len(res.GetUsers()))
	}

	// 50 users case
	for i := 0; i < 50; i++ {
		su := &pb.StoredUser{
			Auth:                   &pb.GramophileAuth{Token: fmt.Sprintf("user-%v", i)},
			User:                   &discogs.User{DiscogsUserId: int32(i)},
			State:                  pb.StoredUser_USER_STATE_IN_WAITLIST,
			ExpectedCollectionSize: 100,
			ExpectedWantlistSize:   50,
			LastItemSyncedTime:     time.Now().Add(-time.Hour * 2).UnixNano(),
		}
		err := d.SaveUser(ctx, su)
		if err != nil {
			t.Fatalf("unable to save user: %v", err)
		}
	}

	res, err = s.GetWaitlistStatus(ctx, &pb.GetWaitlistStatusRequest{})
	if err != nil {
		t.Fatalf("GetWaitlistStatus returned error: %v", err)
	}

	if len(res.GetUsers()) != 50 {
		t.Errorf("Expected 50 users, got %v", len(res.GetUsers()))
	}

	for _, u := range res.GetUsers() {
		if !u.GetIsStuck() {
			t.Errorf("Expected user to be stuck")
		}
		if u.GetFullySynced() {
			t.Errorf("Expected user not to be fully synced")
		}
		if u.GetEtaSeconds() != 150 { // (100 - 0) + (50 - 0)
			t.Errorf("Expected eta 150, got %v", u.GetEtaSeconds())
		}
	}
}
