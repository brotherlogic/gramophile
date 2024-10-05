package integration

import (
	"log"
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

func TestIntent_SinglePush(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{Keep: pb.KeepStatus_MINT_UP_KEEP, MintIds: []int64{12}},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	log.Printf("STARTING2: %v", di.GetCallCount())

	//Run the intent
	qc.FlushQueue(ctx)

	log.Printf("STARTING3: %v", di.GetCallCount())

	resp, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}
	rec := resp.GetRecords()[0].GetRecord()
	if rec.GetKeepStatus() != pb.KeepStatus_MINT_UP_KEEP {
		t.Errorf("Keep was not set: %v", rec)
	}

	// Set something else
	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{NewScore: 5},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	// Should have been 4 Discogs calls (2xgetFields, SetMintUp and SetScore)
	if di.GetCallCount() != 4 {
		t.Errorf("Wrong number of discogs calls (%v)", di.GetCallCount())
	}
}
