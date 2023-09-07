package server

import (
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queue/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSaveAndLoadWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" {
		t.Errorf("Bad list returned: %v", val)
	}
}

func TestUpdateWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" {
		t.Fatalf("Bad list returned: %v", val)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{Name: "testing", AddId: 123})
	if err != nil {
		t.Fatalf("Unable to update wantlist: %v", err)
	}

	val, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" || len(val.List.GetEntries()) != 1 || val.GetList().GetEntries()[0].GetId() != 123 {
		t.Errorf("Bad list returned: %v", val)
	}

}
