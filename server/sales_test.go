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

func TestAddSale(t *testing.T) {
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

	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234}})

	s := Server{d: d, di: di, qc: qc}
	_, err = s.AddSale(ctx, &pb.AddSaleRequest{
		Params: &pbd.SaleParams{
			ReleaseId: 123,
		},
	})
	if err != nil {
		t.Fatalf("Unable to add sale")
	}

	elems, err := qc.List(ctx, &pb.ListRequest{})
	if err != nil {
		t.Fatalf("Unable to list queue elements")
	}
	if len(elems.GetElements()) != 1 {
		t.Errorf("Wrong number of queued elements: %v", len(elems.GetElements()))
	}

}
