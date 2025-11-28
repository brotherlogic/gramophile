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

func TestAdd_FillsInRecordDetails(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{AddConfig: &pb.AddConfig{
			Adds:          pb.Enabled_ENABLED_ENABLED,
			DefaultFolder: "12 Inches"}},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 10, Name: "Purchase Price"},
		{Id: 11, Name: "Purchase Location"}}}
	di.AddCollectionRelease(&pbd.Release{Id: 123, InstanceId: 123, Title: "test-title"})

	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add the record
	_, err = s.AddRecord(ctx, &pb.AddRecordRequest{
		Id:       123,
		Price:    12334,
		Location: "Downhome",
	})
	if err != nil {
		t.Fatalf("Unable to add record: %v", err)
	}

	//Run the intent
	qc.FlushQueue(ctx)

	resp, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		IncludeHistory: true,
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{ReleaseId: 123}}})
	if err != nil {
		t.Fatalf("Bad record retrieve: %v", err)
	}
	rec := resp.GetRecords()[0].GetRecord()
	if rec.GetRelease().GetTitle() != "test-title" {
		t.Errorf("Record was not cached: %v", rec)
	}
}
