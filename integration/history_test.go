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
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestHistoryBackfill(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Width: 12, Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{NoUpdates: true})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Width: 20, Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{NoUpdates: true})
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
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	rec, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		IncludeHistory: true,
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234},
		},
	})
	if err != nil {
		t.Fatalf("Error in getting record: %v", err)
	}

	if len(rec.GetRecords()[0].GetUpdates()) > 0 {
		t.Fatalf("Record already has updates: %v", rec)
	}

	// Run the backfill
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_FanoutHistory{
				FanoutHistory: &pb.FanoutHistory{
					Userid: int64(123),
					Type:   pb.UpdateType_UPDATE_WIDTH,
				},
			},
		},
	})
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Unable to flush queue: %v", err)
	}

	rec, err = s.GetRecord(ctx, &pb.GetRecordRequest{
		IncludeHistory: true,
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234},
		},
	})
	if err != nil {
		t.Fatalf("Error in getting record: %v", err)
	}

	if len(rec.GetRecords()[0].GetUpdates()) == 0 {
		t.Fatalf("Record  has no updates: %v", rec)
	}

}
