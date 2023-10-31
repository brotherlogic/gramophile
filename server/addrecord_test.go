package server

import (
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestAddRecord(t *testing.T) {
	ctx := getTestContext(12345)

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err := s.AddRecord(ctx, &pb.AddRecordRequest{
		Id: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add record: %v", err)
	}

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		Intent:     &pb.Intent{GoalFolder: "12 Inches"},
		InstanceId: 1234,
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	qc.FlushQueue(ctx)

	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{GetRecordWithId: &pb.GetRecordWithId{ReleaseId: 123}},
	})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRecords()[0].GetGoalFolder() != "12 Inches" {
		t.Errorf("Bad record return: %v", r)
	}

}
