package integration

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func buildTestScaffold(t *testing.T) (context.Context, *server.Server, db.Database, *queuelogic.Queue) {
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
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Sales:  []*pbd.SaleItem{}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	return ctx, s, d, qc
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
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
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
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
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
	qc = queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s = server.BuildServer(d, di, qc)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}},
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
	})
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
				Mandate:                pb.Mandate_REQUIRED,
				HandlePriceUpdates:     pb.Mandate_REQUIRED,
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_MINIMAL_REDUCE,
			},
		},
	})
	if err != nil {
		t.Fatalf("unable to set config: %v", err)
	}

	// Run a sale update loop
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
	})
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
				Mandate:                pb.Mandate_REQUIRED,
				HandlePriceUpdates:     pb.Mandate_REQUIRED,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
	})
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
				Mandate:                pb.Mandate_REQUIRED,
				HandlePriceUpdates:     pb.Mandate_REQUIRED,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
	})
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
				Mandate:                      pb.Mandate_REQUIRED,
				HandlePriceUpdates:           pb.Mandate_REQUIRED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
	})
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
				Mandate:                      pb.Mandate_REQUIRED,
				HandlePriceUpdates:           pb.Mandate_REQUIRED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
			if sale.GetCurrentPrice().Value != 3750 {
				t.Errorf("Price was not updated (should be 3750): %v", sale)
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
	})
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
				Mandate:                      pb.Mandate_REQUIRED,
				HandlePriceUpdates:           pb.Mandate_REQUIRED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
	})
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
				Mandate:                      pb.Mandate_REQUIRED,
				HandlePriceUpdates:           pb.Mandate_REQUIRED,
				UpdateFrequencySeconds:       10,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
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
			Auth:  "123",
			Entry: &pb.QueueElement_LinkSales{},
		},
	})
	if err != nil {
		t.Fatalf("Bad enqueue: %v", err)
	}
	err = q.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Bad flush: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:  "123",
			Entry: &pb.QueueElement_RefreshSales{},
		},
	})
	if err != nil {
		t.Fatalf("Unable to enqueue request: %v", err)
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
