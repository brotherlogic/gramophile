package integration

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/server"

	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestMoveApplied(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{
			&pbd.Folder{Name: "Listening Pile", Id: 123},
			&pbd.Folder{Name: "Limbo", Id: 125},
		},
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add a record that needs to be moved
	d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{FolderId: 125, InstanceId: 1234},
		Arrived: time.Now().Unix(),
	})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			CreateFolders: pb.Create_AUTOMATIC,
			CreateMoves:   pb.Create_AUTOMATIC,
			ArrivedConfig: &pb.ArrivedConfig{Mandate: pb.Mandate_REQUIRED},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}
	qc.FlushQueue(ctx)

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecord() == nil || r.GetRecord().GetRelease().GetFolderId() != 123 {
		t.Errorf("Record was not moved: %v", r.GetRecord())
	}
}

func TestRandomMoveHappensPostIntent(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{
			&pbd.Folder{Name: "Listening Pile", Id: 123},
			&pbd.Folder{Name: "Limbo", Id: 125},
		},
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add a record that needs to be moved
	d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{FolderId: 125, InstanceId: 1234},
	})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			CreateFolders: pb.Create_AUTOMATIC,
			CreateMoves:   pb.Create_AUTOMATIC,
			Moves: []*pb.FolderMove{
				{
					Name:       "test-move",
					MoveFolder: "Listening Pile",
					Criteria: &pb.MoveCriteria{
						Arrived: pb.Bool_TRUE,
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}
	qc.FlushQueue(ctx)

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecord() == nil || r.GetRecord().GetRelease().GetFolderId() != 123 {
		t.Errorf("Record was not moved: %v", r.GetRecord())
	}
}
