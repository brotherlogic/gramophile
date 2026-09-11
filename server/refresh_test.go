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

func TestRefreshRelease(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234}})

	s := Server{d: d, di: di, qc: qc}

	_, err = s.RefreshRecord(ctx, &pb.RefreshRecordRequest{
		InstanceId: 1234,
	})
	if err != nil {
		t.Errorf("Unable to refresh Release: %v", err)
	}
}

func TestRefreshReleaseJustState(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234}})

	s := Server{d: d, di: di, qc: qc}

	_, err = s.RefreshRecord(ctx, &pb.RefreshRecordRequest{
		InstanceId: 1234,
		JustState:  true,
	})
	if err != nil {
		t.Errorf("Unable to refresh Release with JustState: %v", err)
	}

	elems, err := qc.List(ctx, &pb.ListRequest{})
	if err != nil {
		t.Fatalf("Unable to list queue elements: %v", err)
	}
	if len(elems.GetElements()) != 2 {
		t.Errorf("Expected 2 queue elements (release + state) when high price is missing, got %v", len(elems.GetElements()))
	}
}

func TestRefreshReleaseJustStateWithHighPrice(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	d.SaveRecord(ctx, 123, &pb.Record{
		Release:   &pbd.Release{InstanceId: 1234},
		HighPrice: &pbd.Price{Value: 5000, Currency: "USD"},
	})

	s := Server{d: d, di: di, qc: qc}

	_, err = s.RefreshRecord(ctx, &pb.RefreshRecordRequest{
		InstanceId: 1234,
		JustState:  true,
	})
	if err != nil {
		t.Errorf("Unable to refresh Release with JustState: %v", err)
	}

	elems, err := qc.List(ctx, &pb.ListRequest{})
	if err != nil {
		t.Fatalf("Unable to list queue elements: %v", err)
	}
	if len(elems.GetElements()) != 1 {
		t.Errorf("Expected 1 queue element (only state) when high price is present, got %v", len(elems.GetElements()))
	}
}

func TestRefreshRecordRecordNotFound(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	_, err = s.RefreshRecord(ctx, &pb.RefreshRecordRequest{
		InstanceId: 9999,
		JustState:  true,
	})
	if err == nil {
		t.Errorf("Expected error when refreshing non-existent record, got nil")
	}
}

func TestRefreshRecordZerothElement(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	_, err = s.RefreshRecord(ctx, &pb.RefreshRecordRequest{
		InstanceId: 0,
		JustState:  true,
	})
	if err == nil {
		t.Errorf("Expected error when refreshing zeroth element, got nil")
	}
}
