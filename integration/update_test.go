package integration

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/server"
	"google.golang.org/grpc/metadata"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	queuelogic "github.com/brotherlogic/gramophile/queue/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(),
		"auth-token",
		fmt.Sprintf("%v", userid))
}

func TestUpdateSavedOnIntentUpdate(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	rec, err := d.GetRecord(ctx, 123, 1234)
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}

	if rec.GetGoalFolder() != "12 Inches" {
		t.Errorf("Goal folder was not set: %v", rec)
	}

	found12InchUpdate := false
	for _, update := range rec.GetUpdates() {
		if update.GetAfter().GetGoalFolder() == "12 Inches" &&
			update.GetBefore().GetGoalFolder() != "12 Inches" {
			found12InchUpdate = true
		}
	}

	if !found12InchUpdate {
		t.Errorf("Updates do not reflect change: %v", rec.GetUpdates())
	}
}
