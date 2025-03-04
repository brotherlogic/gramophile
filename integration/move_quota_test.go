package integration

import (
	"strings"
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

func TestMoveLoopIsCaught(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
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
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add a record that needs to be moved
	d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{FolderId: 125, InstanceId: 1234},
		Arrived: 1234,
	})

	// Setup a move loop
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			CreateFolders: pb.Create_AUTOMATIC,
			CreateMoves:   pb.Create_AUTOMATIC,
			Moves: []*pb.FolderMove{
				{
					Origin:     pb.Create_MANUAL,
					Name:       "test-move-1",
					MoveFolder: "Listening Pile",
					Criteria: &pb.MoveCriteria{
						Arrived: pb.Bool_TRUE,
					},
				},
				{
					Origin:     pb.Create_MANUAL,
					Name:       "test-move-2",
					MoveFolder: "Limbo",
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
		IncludeHistory: true,
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecords()[0].GetRecord() == nil || r.GetRecords()[0].GetRecord().GetRelease().GetFolderId() != 123 {
		t.Errorf("Record was not moved: %v", r.GetRecords()[0].GetRecord())
	}

	count := 0
	for _, move := range r.GetRecords()[0].GetUpdates() {
		for _, exp := range move.GetExplanation() {
			if strings.HasPrefix(exp, "Moved to") {
				count++
			}
		}
	}

	if count < 2 || count > 8 {
		t.Errorf("Too many (or too few) moves [%v] have been made: %v", count, r.GetRecords()[0].GetUpdates())
	}

	user, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Bad user load: %v", err)
	}

	found := false
	for _, fm := range user.User.GetMoves() {
		if fm.GetMoveState() == pb.MoveState_BLOCKED_BECAUSE_OF_LOOP {
			found = true
		}
	}

	if !found {
		t.Errorf("No moves have been blocked because of over quota")
	}
}
