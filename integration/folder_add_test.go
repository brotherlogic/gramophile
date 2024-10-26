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
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestSetGoalFolderCreatesFolder(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			CleaningConfig: &pb.CleaningConfig{Cleaning: pb.Mandate_REQUIRED}, CreateFolders: pb.Create_AUTOMATIC,
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	//Run the queue
	qc.FlushQueue(ctx)

	// Reload the user
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

func TestEnableSalesCreatesFoldersAndMoves(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{Mandate: pb.Mandate_REQUIRED}, CreateFolders: pb.Create_AUTOMATIC, CreateMoves: pb.Create_AUTOMATIC,
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	//Run the queue
	qc.FlushQueue(ctx)

	// Reload the user
	folders, err := di.GetUserFolders(ctx)
	if err != nil {
		t.Fatalf("Unable to get user folders: %v", err)
	}

	found := false
	for _, folder := range folders {
		if folder.GetName() == "For Sale" {
			found = true
		}
	}

	if !found {
		t.Errorf("Folder was not added: %v", folders)
	}

	user, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Unable to retreive user: %v", err)
	}

	if len(user.GetUser().GetMoves()) == 0 {
		t.Errorf("Moves were not added as part of sales")
	}
}
