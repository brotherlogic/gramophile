package integration

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestSetListen_Success(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{Rating: make(map[int64]int32), UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "LastListenDate"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			ListenConfig: &pb.ListenConfig{
				Mandate: pb.Mandate_REQUIRED,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent: &pb.Intent{
			ListenTime: time.Now().UnixNano(),
			NewScore:   5},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	resp, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}
	rec := resp.GetRecords()[0].GetRecord()
	if rec.GetLastListenTime() == 0 || rec.Release.GetRating() != 5 {
		t.Errorf("Listening was not set: %v", rec)
	}
}

func TestSetListen_ResetScore(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{
		InstanceId: 1234,
		FolderId:   12,
		Rating:     5,
		Labels:     []*pbd.Label{{Name: "AAA"}}}})
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
	di := &discogs.TestDiscogsClient{Rating: make(map[int64]int32), UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "LastListenDate"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			ListenConfig: &pb.ListenConfig{
				Mandate: pb.Mandate_REQUIRED,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent: &pb.Intent{
			ListenTime: time.Now().UnixNano(),
			NewScore:   -1},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Error setting intent: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	resp, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}
	rec := resp.GetRecords()[0].GetRecord()
	if rec.GetLastListenTime() == 0 || rec.Release.GetRating() != 0 {
		t.Errorf("Listening was not set (should have listen time and zero rating): %v", rec)
	}
}
