package server

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestGetCollectionStates(t *testing.T) {
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
	s := Server{d: d, di: di, qc: qc}

	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 123, FolderId: 12}})
	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 124, FolderId: 10}})

	stats, err := s.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		t.Fatalf("Unable to get stats: %v", err)
	}

	if stats.GetCollectionStats().GetFolderToCount()[12] != 1 || stats.GetCollectionStats().GetFolderToCount()[10] != 1 {
		t.Errorf("Bad collection stats: %v", stats)
	}
}

func TestGetSaleStats(t *testing.T) {
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
	s := Server{d: d, di: di, qc: qc}

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       12,
		SaleState:    pbd.SaleStatus_SOLD,
		SoldDate:     time.Now().UnixNano(),
		CurrentPrice: &pbd.Price{Value: 123},
	})

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       13,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		SoldDate:     time.Now().UnixNano(),
		CurrentPrice: &pbd.Price{Value: 123},
	})

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       14,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		SoldDate:     time.Now().UnixNano(),
		CurrentPrice: &pbd.Price{Value: 123},
		TimeAtMedian: 10,
	})

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       15,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		SoldDate:     time.Now().UnixNano(),
		CurrentPrice: &pbd.Price{Value: 123},
		TimeAtMedian: 10,
		TimeAtLow:    20,
	})

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       16,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		SoldDate:     time.Now().UnixNano(),
		CurrentPrice: &pbd.Price{Value: 123},
		TimeAtMedian: 10,
		TimeAtLow:    20,
		TimeAtStale:  30,
	})

	stats, err := s.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		t.Fatalf("Unable to get stats: %v", err)
	}

	if stats.GetSaleStats().GetYearTotals()[int32(time.Now().Year())] != 123 {
		t.Errorf("Bad collection stats: %v", stats)
	}

	if stats.GetSaleStats().GetStateCount()["TO_MEDIAN"] != 1 {
		t.Errorf("Bad state count: %v", stats)
	}

	if stats.GetSaleStats().GetStateCount()["TO_LOW"] != 1 {
		t.Errorf("Bad state count: %v", stats)
	}

	if stats.GetSaleStats().GetStateCount()["TO_STALE"] != 1 {
		t.Errorf("Bad state count: %v", stats)
	}

	if stats.GetSaleStats().GetStateCount()["STALE"] != 1 {
		t.Errorf("Bad state count: %v", stats)
	}
}
