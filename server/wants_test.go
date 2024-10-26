package server

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func getTestServer(t *testing.T) (*Server, context.Context) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := &Server{d: d, di: di, qc: qc}

	return s, ctx
}

func TestAddWant_Success(t *testing.T) {
	s, ctx := getTestServer(t)

	_, err := s.AddWant(ctx, &pb.AddWantRequest{WantId: 45})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	val, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wantlsit: %v", err)
	}

	if len(val.GetWants()) != 1 || val.GetWants()[0].GetWant().Id != 45 {
		t.Errorf("Error in returned wants for set: %v", val)
	}
}

func TestAddWant_Failure(t *testing.T) {
	s, _ := getTestServer(t)
	ctx := getTestContext(1234)

	val, err := s.AddWant(ctx, &pb.AddWantRequest{WantId: 45})
	if err == nil {
		t.Fatalf("Should have failed: %v", val)
	}
}

func TestRefreshWant_Success(t *testing.T) {
	s, ctx := getTestServer(t)

	_, err := s.AddWant(ctx, &pb.AddWantRequest{WantId: 45})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	val, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wantlsit: %v", err)
	}

	if len(val.GetWants()) != 1 || val.GetWants()[0].GetWant().Id != 45 {
		t.Errorf("Error in returned wants for set: %v", val)
	}

	_, err = s.RefreshWant(ctx, &pb.RefreshWantRequest{WantId: 45})
	if err != nil {
		t.Fatalf("Unable to refresh want: %v", err)
	}

	val, err = s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wantlsit: %v", err)
	}

	if len(val.GetWants()) != 1 || val.GetWants()[0].GetWant().Id != 45 || val.GetWants()[0].GetWant().GetState() != pb.WantState_WANT_UNKNOWN {
		t.Errorf("Error in returned wants for set: %v", val)
	}

}
