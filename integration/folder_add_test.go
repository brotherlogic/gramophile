package integration

import (
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestSetGoalFolderCreatesFolder(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			CleaningConfig: &pb.CleaningConfig{Cleaning: pb.Mandate_REQUIRED, Create: pb.CreateFolders_AUTOMATIC},
		},
	})

	//Run the queue
	qc.FlushQueue(ctx)

	// Get all the folders
	folders, err := di.GetUserFolders(ctx)
	if err != nil {
		t.Fatalf("Unable to get user folders: %v", err)
	}

	found := false
	for _, folder := range folders {
		if folder.GetName() == "Cleaning Pile" {
			found = true
		}
	}

	if !found {
		t.Errorf("Folder was not added: %v", folders)
	}
}
