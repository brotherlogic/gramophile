package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	pstore_client "github.com/brotherlogic/pstore/client"
	rspb "github.com/brotherlogic/pstore/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

func buildTestScaffold(t *testing.T) (context.Context, *server.Server, db.Database, *queuelogic.Queue) {
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
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales:  []*pbd.SaleItem{}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	return ctx, s, d, qc
}

func TestSyncSales_Success(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Config:  &pb.GramophileConfig{SaleConfig: &pb.SaleConfig{UpdateType: pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE}},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{Status: pbd.SaleStatus_FOR_SALE, ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
		},
	})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if sales.GetRecords()[0].GetRecord().GetRelease().GetId() != 123 {
		t.Fatalf("Bad record returned: %v", sales)
	}

	if sales.GetRecords()[0].GetRecord().GetSaleId() != 12345 {
		t.Errorf("Sale info not returned: %v", sales.GetRecords()[0].GetRecord())
	}

}

func TestSyncSales_DeleteSuccess(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Config:  &pb.GramophileConfig{SaleConfig: &pb.SaleConfig{UpdateType: pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE}},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{Status: pbd.SaleStatus_FOR_SALE, ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
		},
	})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if sales.GetRecords()[0].GetRecord().GetRelease().GetId() != 123 {
		t.Fatalf("Bad record returned: %v", sales)
	}

	if sales.GetRecords()[0].GetRecord().GetSaleId() != 12345 {
		t.Errorf("Sale info not returned pre delete: %v", sales.GetRecords()[0].GetRecord())
	}

	// Now remove the record from the mix
	di = &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{}}
	qc = queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s = server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Force:     true, // Forcing a sale refresh here
			Entry:     &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
		},
	})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err = s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234}}})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if sales.GetRecords()[0].GetRecord().GetRelease().GetId() != 123 {
		t.Fatalf("Bad record returned: %v", sales)
	}

	if sales.GetRecords()[0].GetRecord().GetSaleId() == 12345 {
		t.Errorf("Sale info has not been removed post delete: %v", sales.GetRecords()[0].GetRecord())
	}
}

func TestSalesPriceIsAdjusted(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          123456,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       12,
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		SaleId: 123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_MINIMAL_REDUCE,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 123456 {
			found = true
			if sale.GetCurrentPrice().Value != 1233 {
				t.Errorf("Price was not updated (should be 1233): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedDownToMedian(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          123456,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 1225},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              100,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 123456 {
			found = true
			if sale.GetCurrentPrice().Value != 1225 {
				t.Errorf("Price was not updated (should be 1225): %v", sale)
			}
		}

		if sale.GetTimeAtMedian() == 0 {
			t.Errorf("The time at median was not updated: %v", sale)
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedUpToMedian(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          123456,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              100,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 123456 {
			found = true
			if sale.GetCurrentPrice().Value != 4000 {
				t.Errorf("Price was not updated (should be 4000): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedDownToLowerBound(t *testing.T) {

	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          1836758812,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
		TimeAtMedian:    time.Now().Add(-time.Minute * 50).UnixNano(),
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
				PostMedianTime:               10 * 60, // 10 minutes
				PostMedianReduction:          30000,
				PostMedianReductionFrequency: 30 * 60, // 30 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 1836758812 {
			found = true
			if sale.GetCurrentPrice().Value != 2000 {
				t.Errorf("Price was not updated (should be 2000): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedDownToLowerBoundWithDelay(t *testing.T) {

	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          1836758812,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
		TimeAtMedian:    time.Now().Add(-time.Minute * 50).UnixNano(),
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
				PostMedianTime:               10 * 60, // 10 minutes
				PostMedianReduction:          125,
				PostMedianReductionFrequency: 15 * 60, // 30 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 1836758812 {
			found = true
			if sale.GetCurrentPrice().Value != 3625 {
				t.Errorf("Price was not updated (should be 3625): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedDownToStaticLowerBound(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          1836758812,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
		TimeAtMedian:    time.Now().Add(-time.Minute * 50).UnixNano(),
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   2100,
				PostMedianTime:               10 * 60, // 10 minutes
				PostMedianReduction:          30000,
				PostMedianReductionFrequency: 30 * 60, // 30 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 1836758812 {
			found = true
			if sale.GetCurrentPrice().Value != 2100 {
				t.Errorf("Price was not updated (should be 2100): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSalesPriceIsAdjustedDownBelowMedianOneCycle(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1234, Currency: "USD"},
		SaleId:          1836758812,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
		TimeAtMedian:    time.Now().Add(-time.Minute * 25).UnixNano(),
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   2100,
				PostMedianTime:               10 * 60, // 10 minutes
				PostMedianReduction:          500,
				PostMedianReductionFrequency: 30 * 60, // 30 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 1836758812 {
			found = true
			if sale.GetCurrentPrice().Value != 3500 {
				t.Errorf("Price was not updated (should be 3500): %v", sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestSaleAdjustedDownToStaleLevel(t *testing.T) {

	ctx, s, d, q := buildTestScaffold(t)

	si := &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 2000, Currency: "USD"},
		SaleId:          1836758812,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		ReleaseId:       123,
		TimeAtMedian:    time.Now().Add(-time.Minute * 50).UnixNano(),
		TimeAtLow:       time.Now().Add(-time.Hour * 50).UnixNano(),
	}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveSale(ctx, 123, si)
	if err != nil {
		t.Fatalf("Can't save sale: %v", err)
	}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                          pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:               pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:           10,
				UpdateType:                       pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE,
				Reduction:                        100,
				LowerBoundStrategy:               pb.LowerBoundStrategy_DISCOGS_LOW,
				PostMedianTime:                   10 * 60, // 10 minutes
				PostMedianReduction:              30000,
				PostMedianReductionFrequency:     30 * 60, // 30 minutes
				PostLowTime:                      10 * 60,
				PostLowReduction:                 500,
				PostLowReductionFrequencySeconds: 30 * 60,
				StaleBound:                       500,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry:     &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, u.GetConfig().GetSaleConfig(), u, q.Enqueue)
	if err != nil {
		t.Fatalf("unable to adjust sales: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("Cannot get sales: %v", err)
	}

	if len(sales) != 1 {
		t.Fatalf("Wrong number of sales: %v", sales)
	}

	found := false
	for _, sid := range sales {
		sale, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("Cannot get sale: %v", err)
		}

		if sale.GetSaleId() == 1836758812 {
			found = true
			if sale.GetCurrentPrice().Value != 1500 {
				t.Errorf("Price was not updated (should be 1500), was %v: %v", sale.GetCurrentPrice().GetValue(), sale)
			}
		}
	}

	if !found {
		t.Errorf("Unable to find sale: %v", sales)
	}
}

func TestAddSale(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Unable to save record")
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				ListingStrategy: pb.SaleConfig_LISTING_STRATEGY_MEDIAN,
			},
		},
	})

	_, err = s.AddSale(ctx, &pb.AddSaleRequest{
		Params: &pbd.SaleParams{
			ReleaseId: 123,
			Price:     12.24,
		},
	})
	if err != nil {
		t.Fatalf("Unable to add sale: %v", err)
	}

	// Link sales and valiate that the record has a sale linked
	q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_LinkSales{
				LinkSales: &pb.LinkSales{RefreshId: 12},
			},
		},
	})

	q.FlushQueue(ctx)

	rec, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234},
		},
	})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if len(rec.GetRecords()) == 0 {
		t.Fatalf("Could not find record: %v", rec)
	}

	if rec.GetRecords()[0].GetSaleInfo().GetSaleId() == 0 {
		t.Errorf("Record was not linked to sale: %v", rec)
	}

	if float64(rec.GetRecords()[0].GetSaleInfo().GetCurrentPrice().GetValue()) != 1224 {
		t.Errorf("Record has wrong sale price: %v", rec)
	}
}

func TestAddSale_WithPriceStrategy(t *testing.T) {
	ctx, s, d, q := buildTestScaffold(t)

	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         123,
			InstanceId: 1234,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}}},
		HighPrice:   &pbd.Price{Currency: "USD", Value: 8000},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      123456,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Unable to save record")
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				ListingStrategy: pb.SaleConfig_LISTING_STRATEGY_MEDIAN,
			},
		},
	})

	_, err = s.AddSale(ctx, &pb.AddSaleRequest{
		Params: &pbd.SaleParams{
			ReleaseId: 123,
		},
	})
	if err != nil {
		t.Fatalf("Unable to add sale: %v", err)
	}

	// Link sales and valiate that the record has a sale linked
	q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_LinkSales{
				LinkSales: &pb.LinkSales{RefreshId: 12},
			},
		},
	})

	q.FlushQueue(ctx)

	rec, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 1234},
		},
	})
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if len(rec.GetRecords()) == 0 {
		t.Fatalf("Could not find record: %v", rec)
	}

	if rec.GetRecords()[0].GetSaleInfo().GetSaleId() == 0 {
		t.Errorf("Record was not linked to sale: %v", rec)
	}

	if float64(rec.GetRecords()[0].GetSaleInfo().GetCurrentPrice().GetValue()) != 4000 {
		t.Errorf("Record has wrong sale price: %v", rec)
	}
}

func TestMultiSaleAdjustment_Resilience(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    104,
				ReleaseId: 204,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Set configuration: enable automated price reduction
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   0,
				PostMedianTime:               1,
				PostMedianReduction:          10,
				PostMedianReductionFrequency: 10,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// 1. Sale 101: Corrupt/missing record in DB that triggers unexpected GetSale unmarshal failure
	_, err = pstore.Write(ctx, &rspb.WriteRequest{
		Key:   "gramophile/user/123/sale/101",
		Value: &anypb.Any{Value: []byte("invalid-corrupt-protobuf-bytes")},
	})
	if err != nil {
		t.Fatalf("unable to write corrupt sale to pstore: %v", err)
	}

	// 2. Sale 102: Sale that has already reached low price bound ("Already reached low price for 102")
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         202,
			InstanceId: 2020,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      102,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record 202: %v", err)
	}
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 2000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 2000, Currency: "USD"},
		SaleId:          102,
		ReleaseId:       202,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		TimeAtMedian:    time.Now().Add(-time.Hour * 50).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale 102: %v", err)
	}

	// 3. Sale 103: Sale with missing pricing metadata (no median price)
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 1500, Currency: "USD"},
		SaleId:          103,
		ReleaseId:       203,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
	})
	if err != nil {
		t.Fatalf("unable to save sale 103: %v", err)
	}

	// 4. Sale 104: Valid sale requiring price adjustment (5000 -> 4900)
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         204,
			InstanceId: 2040,
			FolderId:   12,
			Condition:  "Very Good Plus (VG+)",
			Labels:     []*pbd.Label{{Name: "AAA"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 4000},
		SaleId:      104,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record 204: %v", err)
	}
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 4000, Currency: "USD"},
		SaleId:          104,
		ReleaseId:       204,
		LastPriceUpdate: 12,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
	})
	if err != nil {
		t.Fatalf("unable to save sale 104: %v", err)
	}

	// Enqueue AdjustSales task to queue
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test MultiSale Resilient Adjustment",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales task: %v", err)
	}

	// Flush queue: executes AdjustSales and subsequently the enqueued UpdateSale task for Sale 104
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("bad flush queue: %v", err)
	}

	// Assertions:
	// 1. Valid sale (104) should be successfully adjusted and saved to DB with new price 4900
	sale104, err := d.GetSale(ctx, 123, 104)
	if err != nil {
		t.Fatalf("unable to get sale 104 from DB: %v", err)
	}
	if sale104.GetCurrentPrice().GetValue() != 4900 {
		t.Errorf("expected sale 104 price to be adjusted to 4900, got: %v", sale104.GetCurrentPrice().GetValue())
	}

	// 2. Discogs client should reflect the updated price for sale 104
	if len(di.Sales) == 0 || di.Sales[0].GetPrice().GetValue() != 4900 {
		t.Errorf("expected discogs client sale 104 price to be 4900, got: %v", di.Sales)
	}

	// 3. Low-price bounded sale (102) should not have been modified
	sale102, err := d.GetSale(ctx, 123, 102)
	if err != nil {
		t.Fatalf("unable to get sale 102 from DB: %v", err)
	}
	if sale102.GetCurrentPrice().GetValue() != 2000 {
		t.Errorf("expected sale 102 price to remain 2000, got: %v", sale102.GetCurrentPrice().GetValue())
	}

	// 4. Missing metadata sale (103) should not have been modified
	sale103, err := d.GetSale(ctx, 123, 103)
	if err != nil {
		t.Fatalf("unable to get sale 103 from DB: %v", err)
	}
	if sale103.GetCurrentPrice().GetValue() != 1500 {
		t.Errorf("expected sale 103 price to remain 1500, got: %v", sale103.GetCurrentPrice().GetValue())
	}

	// 5. User's LastSaleAdjust timestamp must be updated in DB upon successful completion of the loop
	user, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("unable to get user from DB: %v", err)
	}
	if user.GetLastSaleAdjust() == 0 {
		t.Errorf("expected user LastSaleAdjust timestamp to be non-zero upon completion, got 0")
	}
}

func TestAdjustSales_EndToEndConditionPreservation(t *testing.T) {
	ctx, s, d, qc := buildTestScaffold(t)

	// Save user with SaleConfig
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				UpdateType: pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:  100,
			},
		},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// 1. Save local record with complete condition metadata and pricing
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              501,
			InstanceId:      1501,
			FolderId:        123,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Value: 4000, Currency: "USD"},
		LowPrice:    &pbd.Price{Value: 2000, Currency: "USD"},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("failed to save record: %v", err)
	}

	// 2. AddSale step: Create sale via server
	_, err = s.AddSale(ctx, &pb.AddSaleRequest{
		InstanceId: 1501,
		Params: &pbd.SaleParams{
			ReleaseId:       501,
			Price:           50.00,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
		},
	})
	if err != nil {
		t.Fatalf("AddSale request failed: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue after AddSale failed: %v", err)
	}

	// Verify sale was created and stored in DB with conditions
	sales, err := d.GetSales(ctx, 123)
	if err != nil || len(sales) == 0 {
		t.Fatalf("expected sales to be present in DB: %v", err)
	}
	saleId := sales[0]

	sale, err := d.GetSale(ctx, 123, saleId)
	if err != nil {
		t.Fatalf("failed to get sale %v: %v", saleId, err)
	}
	if sale.GetCondition() != "Near Mint (NM or M-)" || sale.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Fatalf("unexpected conditions on sale after AddSale: condition=%q, sleeve=%q", sale.GetCondition(), sale.GetSleeveCondition())
	}

	// 3. HardLink step: Trigger LinkSales / HardLink and ensure record and sale are linked with conditions intact
	records, err := d.GetRecords(ctx, 123)
	if err != nil {
		t.Fatalf("failed to get records: %v", err)
	}
	var recObjs []*pb.Record
	for _, r := range records {
		recObj, err := d.GetRecord(ctx, 123, r)
		if err != nil {
			t.Fatalf("failed to get record: %v", err)
		}
		recObjs = append(recObjs, recObj)
	}
	var saleObjs []*pb.SaleInfo
	for _, sid := range sales {
		saleObj, err := d.GetSale(ctx, 123, sid)
		if err != nil {
			t.Fatalf("failed to get sale: %v", err)
		}
		saleObjs = append(saleObjs, saleObj)
	}

	bgRunner := background.GetBackgroundRunner(d, "", "", "")
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("failed to get user: %v", err)
	}
	err = bgRunner.HardLink(ctx, u, recObjs, saleObjs)
	if err != nil {
		t.Fatalf("HardLink failed: %v", err)
	}

	linkedSale, err := d.GetSale(ctx, 123, saleId)
	if err != nil {
		t.Fatalf("failed to get linked sale: %v", err)
	}
	if linkedSale.GetCondition() != "Near Mint (NM or M-)" || linkedSale.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Fatalf("unexpected conditions on sale after HardLink: condition=%q, sleeve=%q", linkedSale.GetCondition(), linkedSale.GetSleeveCondition())
	}

	// 4. AdjustSales step: Enqueue AdjustSales and flush queue (executing AdjustSales -> UpdateSale -> ProcessUpdateSale)
	// Reset LastPriceUpdate so it qualifies for adjustment
	linkedSale.LastPriceUpdate = 1
	err = d.SaveSale(ctx, 123, linkedSale)
	if err != nil {
		t.Fatalf("failed to update sale timestamp: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test End-to-End AdjustSales",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("failed to enqueue AdjustSales: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue after AdjustSales failed: %v", err)
	}

	// 5. Assertions:
	// Verify sale in DB has adjusted price towards median (5000 -> 4900)
	adjustedSale, err := d.GetSale(ctx, 123, saleId)
	if err != nil {
		t.Fatalf("failed to get adjusted sale: %v", err)
	}
	if adjustedSale.GetCurrentPrice().GetValue() != 4900 {
		t.Errorf("expected adjusted price 4900, got: %v", adjustedSale.GetCurrentPrice().GetValue())
	}

	// Verify conditions are preserved after price adjustment
	if adjustedSale.GetCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected condition 'Near Mint (NM or M-)' to be preserved, got %q", adjustedSale.GetCondition())
	}
	if adjustedSale.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected sleeve condition 'Very Good Plus (VG+)' to be preserved, got %q", adjustedSale.GetSleeveCondition())
	}
}

func TestAdjustSales_PostMedian_ImmediateFirstCycleReduction(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    1001,
				ReleaseId: 2001,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Set configuration: post-median reduction enabled
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   1000,
				PostMedianTime:               600, // 10 minutes
				PostMedianReduction:          200,
				PostMedianReductionFrequency: 300, // 5 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Save record and sale in DB:
	// TimeAtMedian is 650s ago (PostMedianTime 600s + 50s elapsed).
	// Within first frequency window of 300s -> Floor(50/300) + 1 = 1 cycle.
	// Target price: 5000 - 1 * 200 = 4800.
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              2001,
			InstanceId:      20010,
			FolderId:        12,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 5000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      1001,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1001,
		ReleaseId:       2001,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LowPrice:        &pbd.Price{Value: 2000, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		TimeAtMedian:    time.Now().Add(-650 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale: %v", err)
	}

	// Enqueue AdjustSales task to queue
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test Post-Median Immediate First Cycle",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales: %v", err)
	}

	// Flush queue: executes AdjustSales -> enqueues UpdateSale -> executes ProcessUpdateSale
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	// Assertions:
	sale1001, err := d.GetSale(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("unable to get sale from DB: %v", err)
	}
	if sale1001.GetCurrentPrice().GetValue() != 4800 {
		t.Errorf("expected sale price to be 4800 on first cycle, got: %v", sale1001.GetCurrentPrice().GetValue())
	}
	if sale1001.GetCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected condition 'Near Mint (NM or M-)', got: %v", sale1001.GetCondition())
	}
	if sale1001.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected sleeve condition 'Very Good Plus (VG+)', got: %v", sale1001.GetSleeveCondition())
	}

	// Discogs client updated price
	if len(di.Sales) == 0 || di.Sales[0].GetPrice().GetValue() != 4800 {
		t.Errorf("expected discogs client sale price 4800, got: %v", di.Sales)
	}
}

func TestAdjustSales_PostMedian_MultipleCyclesReduction(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    1002,
				ReleaseId: 2002,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Set configuration: post-median reduction enabled
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   1000,
				PostMedianTime:               600, // 10 minutes
				PostMedianReduction:          250,
				PostMedianReductionFrequency: 300, // 5 minutes
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// TimeAtMedian is 1550s ago (PostMedianTime 600s + 950s elapsed post median).
	// With frequency 300s: Floor(950/300) + 1 = 3 + 1 = 4 cycles.
	// Target price: 5000 - 4 * 250 = 4000.
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              2002,
			InstanceId:      20020,
			FolderId:        12,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 5000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      1002,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1002,
		ReleaseId:       2002,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LowPrice:        &pbd.Price{Value: 2000, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		TimeAtMedian:    time.Now().Add(-1550 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test Post-Median Multiple Cycles",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	sale1002, err := d.GetSale(ctx, 123, 1002)
	if err != nil {
		t.Fatalf("unable to get sale from DB: %v", err)
	}
	if sale1002.GetCurrentPrice().GetValue() != 4000 {
		t.Errorf("expected sale price 4000 after 4 cycles, got: %v", sale1002.GetCurrentPrice().GetValue())
	}
	if len(di.Sales) == 0 || di.Sales[0].GetPrice().GetValue() != 4000 {
		t.Errorf("expected discogs client sale price 4000, got: %v", di.Sales)
	}
}

func TestAdjustSales_PostMedian_RespectsLowerBound_StaticLow(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    1003,
				ReleaseId: 2003,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Set configuration: Static lower bound of 3800
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   3800,
				PostMedianTime:               600,
				PostMedianReduction:          500,
				PostMedianReductionFrequency: 300,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// 2400s elapsed -> 1800s post median -> Floor(1800/300) + 1 = 7 cycles.
	// Target before lower bound: 5000 - 7 * 500 = 1500.
	// Clamped to LowerBound: 3800.
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              2003,
			InstanceId:      20030,
			FolderId:        12,
			Condition:       "Very Good Plus (VG+)",
			SleeveCondition: "Very Good (VG)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 5000},
		SaleId:      1003,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1003,
		ReleaseId:       2003,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		SleeveCondition: "Very Good (VG)",
		TimeAtMedian:    time.Now().Add(-2400 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test Post-Median Static Lower Bound",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	sale1003, err := d.GetSale(ctx, 123, 1003)
	if err != nil {
		t.Fatalf("unable to get sale from DB: %v", err)
	}
	if sale1003.GetCurrentPrice().GetValue() != 3800 {
		t.Errorf("expected sale price clamped to static lower bound 3800, got: %v", sale1003.GetCurrentPrice().GetValue())
	}
	if len(di.Sales) == 0 || di.Sales[0].GetPrice().GetValue() != 3800 {
		t.Errorf("expected discogs client sale price 3800, got: %v", di.Sales)
	}
}

func TestAdjustSales_PostMedian_RespectsLowerBound_DiscogsLow(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    1004,
				ReleaseId: 2004,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Set configuration: Discogs Low strategy
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
				PostMedianTime:               600,
				PostMedianReduction:          500,
				PostMedianReductionFrequency: 300,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// 2400s elapsed -> 1800s post median -> Floor(1800/300) + 1 = 7 cycles.
	// Target before lower bound: 5000 - 7 * 500 = 1500.
	// Clamped to Discogs Low: 3200.
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              2004,
			InstanceId:      20040,
			FolderId:        12,
			Condition:       "Very Good Plus (VG+)",
			SleeveCondition: "Very Good (VG)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 5000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 3200},
		SaleId:      1004,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1004,
		ReleaseId:       2004,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LowPrice:        &pbd.Price{Value: 3200, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		SleeveCondition: "Very Good (VG)",
		TimeAtMedian:    time.Now().Add(-2400 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test Post-Median Discogs Low Bound",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	sale1004, err := d.GetSale(ctx, 123, 1004)
	if err != nil {
		t.Fatalf("unable to get sale from DB: %v", err)
	}
	if sale1004.GetCurrentPrice().GetValue() != 3200 {
		t.Errorf("expected sale price clamped to Discogs low 3200, got: %v", sale1004.GetCurrentPrice().GetValue())
	}
	if len(di.Sales) == 0 || di.Sales[0].GetPrice().GetValue() != 3200 {
		t.Errorf("expected discogs client sale price 3200, got: %v", di.Sales)
	}
	if sale1004.GetTimeAtLow() == 0 {
		t.Errorf("expected TimeAtLow timestamp to be set on sale reaching low price")
	}
}

func TestAdjustSales_PostMedian_HoldingBeforePostMedianTime(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales: []*pbd.SaleItem{
			{
				SaleId:    1005,
				ReleaseId: 2005,
				Status:    pbd.SaleStatus_FOR_SALE,
				Price:     &pbd.Price{Value: 5000, Currency: "USD"},
			},
		},
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   1000,
				PostMedianTime:               600, // 10 minutes
				PostMedianReduction:          200,
				PostMedianReductionFrequency: 300,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// TimeAtMedian is only 300s ago (< 600s PostMedianTime).
	// Should hold price at median (5000).
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:              2005,
			InstanceId:      20050,
			FolderId:        12,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			Labels:          []*pbd.Label{{Name: "TestLabel"}},
		},
		MedianPrice: &pbd.Price{Currency: "USD", Value: 5000},
		LowPrice:    &pbd.Price{Currency: "USD", Value: 2000},
		SaleId:      1005,
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("unable to save record: %v", err)
	}

	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1005,
		ReleaseId:       2005,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LowPrice:        &pbd.Price{Value: 2000, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		TimeAtMedian:    time.Now().Add(-300 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("unable to save sale: %v", err)
	}

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test Post-Median Holding State",
			Auth:      "123",
			Entry: &pb.QueueElement_AdjustSales{
				AdjustSales: &pb.AdjustSales{},
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to enqueue AdjustSales: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	sale1005, err := d.GetSale(ctx, 123, 1005)
	if err != nil {
		t.Fatalf("unable to get sale from DB: %v", err)
	}
	if sale1005.GetCurrentPrice().GetValue() != 5000 {
		t.Errorf("expected sale price to hold at 5000 before PostMedianTime, got: %v", sale1005.GetCurrentPrice().GetValue())
	}
}

func TestAdjustSales_PostMedian_EnqueuedUpdateSalePayload(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test-auth-token-123"},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				Enabled:                      pb.Enabled_ENABLED_ENABLED,
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
				LowerBound:                   1000,
				PostMedianTime:               600,
				PostMedianReduction:          200,
				PostMedianReductionFrequency: 300,
			},
		},
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// 750s elapsed since median -> 150s post median -> 1 cycle (target 5000 - 200 = 4800)
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1006,
		ReleaseId:       2006,
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		MedianPrice:     &pbd.Price{Value: 5000, Currency: "USD"},
		LowPrice:        &pbd.Price{Value: 2000, Currency: "USD"},
		LastPriceUpdate: 1,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		TimeAtMedian:    time.Now().Add(-750 * time.Second).UnixNano(),
	})
	if err != nil {
		t.Fatalf("failed to save sale: %v", err)
	}

	var capturedRequests []*pb.EnqueueRequest
	enqueueInterceptor := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		capturedRequests = append(capturedRequests, req)
		return &pb.EnqueueResponse{}, nil
	}

	b := background.GetBackgroundRunner(d, "", "", "")
	err = b.AdjustSales(ctx, user.GetConfig().GetSaleConfig(), user, enqueueInterceptor)
	if err != nil {
		t.Fatalf("AdjustSales failed: %v", err)
	}

	if len(capturedRequests) != 1 {
		t.Fatalf("expected 1 enqueued request, got %v", len(capturedRequests))
	}

	req := capturedRequests[0]
	if req.GetElement().GetAuth() != "test-auth-token-123" {
		t.Errorf("expected Auth 'test-auth-token-123', got %v", req.GetElement().GetAuth())
	}

	updateSale := req.GetElement().GetUpdateSale()
	if updateSale == nil {
		t.Fatalf("expected UpdateSale payload in queue element, got nil")
	}

	if updateSale.GetSaleId() != 1006 {
		t.Errorf("expected SaleId 1006, got %v", updateSale.GetSaleId())
	}
	if updateSale.GetReleaseId() != 2006 {
		t.Errorf("expected ReleaseId 2006, got %v", updateSale.GetReleaseId())
	}
	if updateSale.GetNewPrice() != 4800 {
		t.Errorf("expected NewPrice 4800, got %v", updateSale.GetNewPrice())
	}
	if updateSale.GetCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected Condition 'Near Mint (NM or M-)', got %q", updateSale.GetCondition())
	}
	if updateSale.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected SleeveCondition 'Very Good Plus (VG+)', got %q", updateSale.GetSleeveCondition())
	}
	if !strings.Contains(updateSale.GetMotivation(), "reducing post median") {
		t.Errorf("expected Motivation to mention 'reducing post median', got %q", updateSale.GetMotivation())
	}
}



