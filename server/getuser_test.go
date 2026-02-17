package server

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestUpgradeUser(t *testing.T) {
	ctx := getTestContext(123)
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123, Username: "david"},
		State:   pb.StoredUser_USER_STATE_IN_WAITLIST,
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	user, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}
	if user.GetUser().GetState() != pb.StoredUser_USER_STATE_IN_WAITLIST {
		t.Errorf("User state was not set: %v", user.GetUser())
	}

	_, err = s.UpgradeUser(context.Background(), &pb.UpgradeUserRequest{Username: "david", NewState: pb.StoredUser_USER_STATE_LIVE})
	if err != nil {
		t.Fatalf("Unable to get upgrade user: %v", err)
	}

	user, err = s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}
	if user.GetUser().GetState() != pb.StoredUser_USER_STATE_LIVE {
		t.Errorf("User state was not set: %v", user.GetUser())
	}
}
