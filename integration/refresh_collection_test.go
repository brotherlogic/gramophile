package integration

import (
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestMultiAddRefreshCollection(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	//s := server.BuildServer(d, di, qc)

	// Run a full update
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshCollection{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue first collection refresh: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshCollection{},
		},
	})
	if err == nil {
		t.Fatalf("Was able to enqueue second collection refresh: %v", err)
	}

}
