package server

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queue_client "github.com/brotherlogic/gramophile/queue_client"
	pstore_client "github.com/brotherlogic/pstore/client"
)

type failDiscogsClient struct {
	*discogs.TestDiscogsClient
	fail bool
}

func (f *failDiscogsClient) HandleDiscogsResponse(ctx context.Context, secret, token, verifier string) (string, string, error) {
	if f.fail {
		return "", "", fmt.Errorf("forced error")
	}
	return "returned_token", "returned_secret", nil
}

func TestServeHTTP_DiscogsError(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &failDiscogsClient{
		TestDiscogsClient: &discogs.TestDiscogsClient{},
		fail:              true,
	}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	// Add an active login attempt
	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "test_token",
				Secret:       "test_secret",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}
	err := d.SaveLogins(ctx, attempts)
	if err != nil {
		t.Fatalf("Failed to save logins: %v", err)
	}

	req := httptest.NewRequest("GET", "/callback?oauth_token=test_token&oauth_verifier=verifier", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %v, got %v", http.StatusInternalServerError, rr.Code)
	}
}

func TestServeHTTP_TokenNotFound(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &failDiscogsClient{
		TestDiscogsClient: &discogs.TestDiscogsClient{},
		fail:              false,
	}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	// Add a login attempt with a different token
	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "another_token",
				Secret:       "test_secret",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}
	err := d.SaveLogins(ctx, attempts)
	if err != nil {
		t.Fatalf("Failed to save logins: %v", err)
	}

	req := httptest.NewRequest("GET", "/callback?oauth_token=not_found_token&oauth_verifier=verifier", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("Expected status code %v, got %v", http.StatusBadRequest, rr.Code)
	}
}

func TestServeHTTP_Success(t *testing.T) {
	ctx := context.Background()
	d := db.NewTestDB(pstore_client.GetTestClient())
	di := &failDiscogsClient{
		TestDiscogsClient: &discogs.TestDiscogsClient{},
		fail:              false,
	}
	qc := queue_client.GetTestClient()

	s := Server{d: d, di: di, qc: qc}

	attempts := &pb.UserLoginAttempts{
		Attempts: []*pb.UserLoginAttempt{
			{
				RequestToken: "test_token",
				Secret:       "test_secret",
				DateAdded:    time.Now().UnixNano(),
			},
		},
	}
	err := d.SaveLogins(ctx, attempts)
	if err != nil {
		t.Fatalf("Failed to save logins: %v", err)
	}

	req := httptest.NewRequest("GET", "/callback?oauth_token=test_token&oauth_verifier=verifier", nil)
	rr := httptest.NewRecorder()

	s.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status code %v, got %v", http.StatusOK, rr.Code)
	}

	loaded, err := d.LoadLogins(ctx)
	if err != nil {
		t.Fatalf("Failed to load logins: %v", err)
	}

	found := false
	for _, login := range loaded.GetAttempts() {
		if login.RequestToken == "test_token" {
			found = true
			if login.UserToken != "returned_token" {
				t.Errorf("Expected UserToken 'returned_token', got '%v'", login.UserToken)
			}
			if login.UserSecret != "returned_secret" {
				t.Errorf("Expected UserSecret 'returned_secret', got '%v'", login.UserSecret)
			}
		}
	}

	if !found {
		t.Errorf("Expected login attempt to still exist and be updated")
	}
}
