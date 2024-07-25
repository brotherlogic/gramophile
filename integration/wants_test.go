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
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestAddMasterWant_WithFilteringt(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	di.AddCollectionRelease(&pbd.Release{Id: 12, MasterId: 123, Formats: []*pbd.Format{{Name: "12 Inch"}}})
	di.AddCollectionRelease(&pbd.Release{Id: 13, MasterId: 123})

	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.AddWant(ctx, &pb.AddWantRequest{
		MasterWantId: 123,
		Filter: &pb.WantFilter{
			Formats: []string{"12 Inch"},
		},
	})

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: 1,
			Entry:   &pb.QueueElement_SyncWants{},
		},
	})

	qc.FlushQueue(ctx)

	if err != nil {
		t.Fatalf("Unable to add want: %v", err)
	}

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Error in getting wants: %v", err)
	}

	if len(wants.GetWants()) != 1 {
		t.Errorf("There should be 1 wants, there's only %v", len(wants.GetWants()))
	}

}

func TestAddMasterWant(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	di.AddCollectionRelease(&pbd.Release{Id: 12, MasterId: 123})
	di.AddCollectionRelease(&pbd.Release{Id: 13, MasterId: 123})

	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.AddWant(ctx, &pb.AddWantRequest{
		MasterWantId: 123,
	})

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: 1,
			Entry:   &pb.QueueElement_SyncWants{},
		},
	})

	qc.FlushQueue(ctx)

	if err != nil {
		t.Fatalf("Unable to add want: %v", err)
	}

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Error in getting wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Errorf("There should be 2 wants, there's only %v -> %v", len(wants.GetWants()), wants.GetWants())
	}

}
