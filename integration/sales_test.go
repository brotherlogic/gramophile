package integration

import (
	"context"
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

func buildTestScaffold(t *testing.T) (context.Context, *server.Server, db.Database) {
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
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	return ctx, s, d
}

func TestSyncSales_Success(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
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
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	sales, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if sales.GetRecord().GetRelease().GetId() != 123 {
		t.Fatalf("Bad record returned: %v", sales)
	}
}

func TestSalesPriceIsAdjusted(t *testing.T) {
	ctx, s, d := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice: &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:       123456,
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		SaleInfo: si,
	})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Mandate:                pb.Mandate_REQUIRED,
				HandlePriceUpdates:     pb.Mandate_REQUIRED,
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_MINIMAL_REDUCE,
			},
		},
	})

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales")
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 123456 {
			found = true
			if sale.GetCurrentPrice().Value != 1235 {
				t.Errorf("Price was not updated (should be 1235): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}
