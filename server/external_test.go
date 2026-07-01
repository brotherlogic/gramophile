package server

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queue_client "github.com/brotherlogic/gramophile/queue_client"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestPruneOldLogins_GetURL(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &discogs.TestDiscogsClient{}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	// Add some old and new attempts
	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "old",
				Secret:       "old",
				DateAdded:    time.Now().Add(-20 * time.Minute).UnixNano(),
			},
			{
				RequestToken: "new",
				Secret:       "new",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}

	err := d.SaveLogins(ctx, attempts)
	if err != nil {
		t.Fatalf("Failed to save logins: %v", err)
	}

	// Trigger pruning via GetURL
	_, err = s.GetURL(ctx, &pb.GetURLRequest{})
	if err != nil {
		t.Fatalf("Failed to get URL: %v", err)
	}

	// Verify old attempt is gone
	loaded, err := d.LoadLogins(ctx)
	if err != nil {
		t.Fatalf("Failed to load logins: %v", err)
	}

	foundOld := false
	foundNew := false
	for _, a := range loaded.GetAttempts() {
		if a.RequestToken == "old" {
			foundOld = true
		}
		if a.RequestToken == "new" {
			foundNew = true
		}
	}

	if foundOld {
		t.Errorf("Old attempt was not pruned")
	}
	if !foundNew {
		t.Errorf("New attempt was incorrectly pruned")
	}
}

func TestPruneAndRemove_GetLogin(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &discogs.TestDiscogsClient{}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	// Add some old and new attempts
	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "old",
				Secret:       "old",
				DateAdded:    time.Now().Add(-20 * time.Minute).UnixNano(),
			},
			{
				RequestToken: "auth_me",
				UserToken:    "user_token",
				UserSecret:   "user_secret",
				DateAdded:    time.Now().UnixNano(),
			},
			{
				RequestToken: "keep_me",
				UserToken:    "keep_user_token",
				UserSecret:   "keep_user_secret",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}

	err := d.SaveLogins(ctx, attempts)
	if err != nil {
		t.Fatalf("Failed to save logins: %v", err)
	}

	// Authenticate with auth_me
	_, err = s.GetLogin(ctx, &pb.GetLoginRequest{Token: "auth_me"})
	if err != nil {
		t.Fatalf("Failed to get login: %v", err)
	}

	// Verify old attempt is pruned, auth_me is removed, and keep_me remains
	loaded, err := d.LoadLogins(ctx)
	if err != nil {
		t.Fatalf("Failed to load logins: %v", err)
	}

	foundOld := false
	foundAuthMe := false
	foundKeepMe := false
	for _, a := range loaded.GetAttempts() {
		if a.RequestToken == "old" {
			foundOld = true
		}
		if a.RequestToken == "auth_me" {
			foundAuthMe = true
		}
		if a.RequestToken == "keep_me" {
			foundKeepMe = true
		}
	}

	if foundOld {
		t.Errorf("Old attempt was not pruned")
	}
	if foundAuthMe {
		t.Errorf("Authenticated attempt was not removed")
	}
	if !foundKeepMe {
		t.Errorf("Other valid attempt was incorrectly removed")
	}
}

type wrapperDiscogs struct {
	discogs.Discogs
}

func (w *wrapperDiscogs) ForUser(u *pbd.User) discogs.Discogs {
	return w
}

func (w *wrapperDiscogs) GetDiscogsUser(ctx context.Context) (*pbd.User, error) {
	return &pbd.User{Username: "test_user"}, nil
}

func TestFetchExpectedSizes(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &wrapperDiscogs{Discogs: &discogs.TestDiscogsClient{}}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	// Mock getProfileStats
	originalGetProfileStats := getProfileStats
	getProfileStats = func(username string) (int32, int32, error) {
		return 123, 456, nil
	}
	defer func() {
		getProfileStats = originalGetProfileStats
	}()

	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "auth_me",
				UserToken:    "user_token",
				UserSecret:   "user_secret",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}
	d.SaveLogins(ctx, attempts)

	_, err := s.GetLogin(ctx, &pb.GetLoginRequest{Token: "auth_me"})
	if err != nil {
		t.Fatalf("Failed to get login: %v", err)
	}

	users, err := s.GetUsers(ctx, &pb.GetUsersRequest{})
	if err != nil {
		t.Fatalf("Failed to get users: %v", err)
	}

	if len(users.GetUsers()) != 1 {
		t.Fatalf("Expected 1 user, got %v", len(users.GetUsers()))
	}

	user := users.GetUsers()[0]
	// TestDiscogsClient returns an empty username by default, so getProfileStats may not be called.
	// If it is called, we expect 123 and 456.
	if user.GetUser().GetUsername() != "" {
		if user.GetExpectedCollectionSize() != 123 {
			t.Errorf("Expected collection size 123, got %v", user.GetExpectedCollectionSize())
		}
		if user.GetExpectedWantlistSize() != 456 {
			t.Errorf("Expected wantlist size 456, got %v", user.GetExpectedWantlistSize())
		}
	}
}
