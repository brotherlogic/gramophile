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

func TestRecordUpdatedPostSync(t *testing.T) {
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

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, MasterId: 12345, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	/*err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 124, InstanceId: 555, MasterId: 12345, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}*/

	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	di.AddCollectionRelease(&pbd.Release{Id: 123, InstanceId: 1234, MasterId: 12345, FolderId: 12, ReleaseDate: 12345678, Labels: []*pbd.Label{{Name: "AAA"}}})
	di.AddNonCollectionRelease(&pbd.Release{Id: 124, InstanceId: 555, MasterId: 12345, ReleaseDate: 1234, Labels: []*pbd.Label{{Name: "AAA"}}})
	di.AddNonCollectionRelease(&pbd.Release{Id: 125, ReleaseDate: 12345678})

	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Run a full update
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshCollection{},
		},
	})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	// Run a notherfull update in order to update the other record
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshCollection{},
		},
	})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	// Check that the colleciton is just one record
	rec, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 555}}})
	if err == nil {
		t.Fatalf("Returned non-collection record: %v", rec)
	}

	// Get the record
	rec, err = s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})

	if err != nil {
		t.Fatalf("Unable to get release: %v", err)
	}

	if rec.GetRecords()[0].GetRecord().GetRelease().GetReleaseDate() != 12345678 {
		t.Errorf("Record was not updated: %v", rec.GetRecords()[0].GetRecord())
	}

	if rec.GetRecords()[0].GetRecord().GetEarliestReleaseDate() != 1234 {
		t.Errorf("Record earliest release date was not updated: %v", rec.GetRecords()[0].GetRecord())
	}
}
