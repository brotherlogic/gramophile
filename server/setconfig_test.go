package server

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queue/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestConfigUpdate_UpdatesTime(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	c1, err := s.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		t.Fatalf("Unable to get state: %v", err)
	}

	nconfig := &pb.GramophileConfig{Basis: pb.Basis_GRAMOPHILE}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("Bad initial config set: %v", err)
	}

	c2, err := s.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		t.Fatalf("Unable to get state: %v", err)
	}

	if c1.GetLastConfigUpdate() == c2.GetLastConfigUpdate() {
		t.Errorf("Collection sync time was not updated: %v (%v)", c2.GetLastConfigUpdate(), time.Unix(c2.GetLastConfigUpdate(), 0))
	}
}

func TestConfigUpdate_FailsOnMissingField(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	nconfig := &pb.GramophileConfig{Basis: pb.Basis_GRAMOPHILE, GoalFolderConfig: &pb.GoalFolderConfig{Mandate: pb.Mandate_REQUIRED}}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err == nil {
		t.Errorf("Set config should have failed on missing field")
	}
}
