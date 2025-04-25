package server

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

func TestAdd_Success(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB(pstore_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{AddConfig: &pb.AddConfig{
			AllowAdds:     pb.Mandate_REQUIRED,
			DefaultFolder: "limbo",
		}},
		Folders: []*pbd.Folder{{Name: "limbo", Id: 123}},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 10, Name: "Goal Folder"},
		{Id: 11, Name: "Purchase Location"},
		{Id: 12, Name: "Purchase Price"},
	}}
	pstore := pstore_client.GetTestClient()
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

	s := Server{d: d, di: di, qc: qc}

	val, err := s.AddRecord(ctx, &pb.AddRecordRequest{
		Id:       123,
		Price:    1234,
		Location: "online",
	})

	if err != nil {
		t.Fatalf("Unable to add record: %v", err)
	}

	// Flush out the queue
	qc.FlushQueue(ctx)

	rec, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: val.GetInstanceId()},
		},
	})

	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if rec.GetRecords()[0].GetRecord().GetPurchaseLocation() != "online" {
		t.Errorf("Record has not been added correctly: %v", rec)
	}
}
