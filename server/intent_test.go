package server

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestGoalFolderAddsIntent_Success(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}}, User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	s := Server{d: d, di: &discogs.TestDiscogsClient{}}

	_, err = s.SetIntent(context.Background(), &pb.SetIntentRequest{
		Intent: &pb.Intent{GoalFolder: "12 Inches"},
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	q := queuelogic.GetTestQueue(rstore)
	q.FlushQueue(ctx)

	rec, err := d.GetRecord(ctx, 123, 1234)
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}

	if rec.GetGoalFolder() != "12 Inches" {
		t.Errorf("Goal folder was not set: %v", rec)
	}
}

func TestGoalFolderAddsIntent_FailMissingFolder(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	s := Server{d: d, di: &discogs.TestDiscogsClient{}}

	_, err = s.SetIntent(context.Background(), &pb.SetIntentRequest{
		Intent: &pb.Intent{GoalFolder: "12 Inches"},
	})
	if err == nil {
		t.Errorf("Intent did not fail with missing folder")
	}

}
