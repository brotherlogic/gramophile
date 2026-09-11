package integration

import (
	"fmt"
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
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestSyncOrders_EndToEndPipeline(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	initialSyncTime := time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC).UnixNano()
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:          &pbd.User{DiscogsUserId: 123},
		Auth:          &pb.GramophileAuth{Token: "123"},
		LastOrderSync: initialSyncTime,
	})
	if err != nil {
		t.Fatalf("Failed to save initial user: %v", err)
	}

	// Pre-create two records in the user's collection
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         2001,
			InstanceId: 10001,
			FolderId:   123,
			Title:      "Existing Record 1",
		},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save record 1: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         2002,
			InstanceId: 10002,
			FolderId:   123,
			Title:      "Existing Record 2",
		},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save record 2: %v", err)
	}

	// Pre-create sale 3001 as FOR_SALE in DB
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       3001,
		ReleaseId:    2001,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
		Condition:    "Very Good (VG)",
	})
	if err != nil {
		t.Fatalf("Failed to save sale 3001: %v", err)
	}

	// Setup mock Discogs with an order containing items 3001 (existing) and 3002 (new)
	orderCreated := time.Date(2023, 8, 15, 12, 0, 0, 0, time.UTC).Unix()
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "LastSaleUpdate"}},
		Orders: []*pbd.Order{
			{
				Id:      "order-12345",
				Status:  "Payment Received",
				Created: orderCreated,
				Items: []*pbd.OrderItem{
					{
						Id:              3001,
						ReleaseId:       2001,
						Price:           &pbd.Price{Value: 1500, Currency: "USD"},
						Condition:       "Near Mint (NM or M-)",
						SleeveCondition: "Near Mint (NM or M-)",
					},
					{
						Id:              3002,
						ReleaseId:       2002,
						Price:           &pbd.Price{Value: 2500, Currency: "USD"},
						Condition:       "Very Good Plus (VG+)",
						SleeveCondition: "Very Good (VG)",
					},
				},
			},
		},
	}

	b := background.GetBackgroundRunner(d, "", "", "")
	qc := queuelogic.GetQueue(pstore, b, di, d)
	s := server.BuildServer(d, di, qc)

	// Enqueue SyncOrders queue element
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "test-orders-pipeline",
			Auth:      "123",
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{
					Page: 1,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue SyncOrders: %v", err)
	}

	// Flush queue to process SyncOrders and the cascaded LinkSales task
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	// 1. Verify existing sale 3001 transitioned to SOLD and price was updated
	sale1, err := d.GetSale(ctx, 123, 3001)
	if err != nil {
		t.Fatalf("Failed to get sale 3001: %v", err)
	}
	if sale1.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected sale 3001 state SOLD, got %v", sale1.GetSaleState())
	}
	expectedSoldDate := time.Unix(orderCreated, 0).UnixNano()
	if sale1.GetSoldDate() != expectedSoldDate {
		t.Errorf("Expected sale 3001 SoldDate %v, got %v", expectedSoldDate, sale1.GetSoldDate())
	}
	if sale1.GetCurrentPrice().GetValue() != 1500 {
		t.Errorf("Expected sale 3001 price 1500, got %v", sale1.GetCurrentPrice().GetValue())
	}

	// 2. Verify new sale 3002 was created as SOLD
	sale2, err := d.GetSale(ctx, 123, 3002)
	if err != nil {
		t.Fatalf("Failed to get sale 3002: %v", err)
	}
	if sale2.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected sale 3002 state SOLD, got %v", sale2.GetSaleState())
	}
	if sale2.GetSoldDate() != expectedSoldDate {
		t.Errorf("Expected sale 3002 SoldDate %v, got %v", expectedSoldDate, sale2.GetSoldDate())
	}
	if sale2.GetCurrentPrice().GetValue() != 2500 {
		t.Errorf("Expected sale 3002 price 2500, got %v", sale2.GetCurrentPrice().GetValue())
	}

	// 3. Verify user.LastOrderSync progressed forward
	user, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.GetLastOrderSync() <= initialSyncTime {
		t.Errorf("Expected user LastOrderSync > %v, got %v", initialSyncTime, user.GetLastOrderSync())
	}

	// 4. Verify LinkSales linked the sales to the records in DB
	rec1, err := d.GetRecord(ctx, 123, 10001)
	if err != nil {
		t.Fatalf("Failed to get record 10001: %v", err)
	}
	if rec1.GetSaleId() != 3001 {
		t.Errorf("Expected record 10001 SaleId 3001, got %v", rec1.GetSaleId())
	}

	rec2, err := d.GetRecord(ctx, 123, 10002)
	if err != nil {
		t.Fatalf("Failed to get record 10002: %v", err)
	}
	if rec2.GetSaleId() != 3002 {
		t.Errorf("Expected record 10002 SaleId 3002, got %v", rec2.GetSaleId())
	}

	// 5. Verify server.GetRecord serves the records with linked SaleInfo
	resp1, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{InstanceId: 10001},
		},
	})
	if err != nil {
		t.Fatalf("Server GetRecord failed for 10001: %v", err)
	}
	if len(resp1.GetRecords()) == 0 {
		t.Fatalf("Server GetRecord returned no records for 10001")
	}
	if resp1.GetRecords()[0].GetRecord().GetSaleId() != 3001 {
		t.Errorf("Expected server record SaleId 3001, got %v", resp1.GetRecords()[0].GetRecord().GetSaleId())
	}
	if resp1.GetRecords()[0].GetSaleInfo().GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected server SaleInfo state SOLD, got %v", resp1.GetRecords()[0].GetSaleInfo().GetSaleState())
	}
}

func TestSyncOrders_LookbackBehavior(t *testing.T) {
	ctx := getTestContext(123)

	t.Run("ColdStart_30DayLookback", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		d := db.NewTestDB(pstore)

		// Cold start: LastOrderSync == 0
		err := d.SaveUser(ctx, &pb.StoredUser{
			Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
			User:          &pbd.User{DiscogsUserId: 123},
			Auth:          &pb.GramophileAuth{Token: "123"},
			LastOrderSync: 0,
		})
		if err != nil {
			t.Fatalf("Failed to save user: %v", err)
		}

		now := time.Now()
		orderOlderThan30Days := now.Add(-40 * 24 * time.Hour).Unix()
		orderWithin30Days := now.Add(-10 * 24 * time.Hour).Unix()

		di := &discogs.TestDiscogsClient{
			UserId: 123,
			Orders: []*pbd.Order{
				{
					Id:      "order-old-40days",
					Status:  "Payment Received",
					Created: orderOlderThan30Days,
					Items: []*pbd.OrderItem{
						{
							Id:        4001,
							ReleaseId: 5001,
							Price:     &pbd.Price{Value: 1000, Currency: "USD"},
						},
					},
				},
				{
					Id:      "order-recent-10days",
					Status:  "Payment Received",
					Created: orderWithin30Days,
					Items: []*pbd.OrderItem{
						{
							Id:        4002,
							ReleaseId: 5002,
							Price:     &pbd.Price{Value: 2000, Currency: "USD"},
						},
					},
				},
			},
		}

		b := background.GetBackgroundRunner(d, "", "", "")
		qc := queuelogic.GetQueue(pstore, b, di, d)

		_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: "test-cold-start-lookback",
				Auth:      "123",
				Entry: &pb.QueueElement_SyncOrders{
					SyncOrders: &pb.SyncOrders{Page: 1},
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}

		err = qc.FlushQueue(ctx)
		if err != nil {
			t.Fatalf("FlushQueue failed: %v", err)
		}

		// Item within 30 days should be synced
		saleRecent, err := d.GetSale(ctx, 123, 4002)
		if err != nil {
			t.Fatalf("Failed to get recent sale: %v", err)
		}
		if saleRecent.GetSaleState() != pbd.SaleStatus_SOLD {
			t.Errorf("Expected recent sale to be SOLD, got %v", saleRecent.GetSaleState())
		}

		// Item older than 30 days should not be synced
		_, err = d.GetSale(ctx, 123, 4001)
		if status.Code(err) != codes.NotFound {
			t.Errorf("Expected 4001 not found (outside 30-day window), got %v", err)
		}

		// User LastOrderSync should now be non-zero
		u, err := d.GetUser(ctx, "123")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		if u.GetLastOrderSync() == 0 {
			t.Errorf("Expected user LastOrderSync to be updated from 0")
		}
	})

	t.Run("SubsequentSync_IncrementalLookback", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		d := db.NewTestDB(pstore)

		// Subsequent sync: LastOrderSync set to 5 days ago
		now := time.Now()
		syncCutoff := now.Add(-5 * 24 * time.Hour)
		err := d.SaveUser(ctx, &pb.StoredUser{
			Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
			User:          &pbd.User{DiscogsUserId: 123},
			Auth:          &pb.GramophileAuth{Token: "123"},
			LastOrderSync: syncCutoff.UnixNano(),
		})
		if err != nil {
			t.Fatalf("Failed to save user: %v", err)
		}

		orderBeforeCutoff := syncCutoff.Add(-2 * 24 * time.Hour).Unix() // 7 days ago
		orderAfterCutoff := syncCutoff.Add(3 * 24 * time.Hour).Unix()   // 2 days ago

		di := &discogs.TestDiscogsClient{
			UserId: 123,
			Orders: []*pbd.Order{
				{
					Id:      "order-before-cutoff",
					Status:  "Payment Received",
					Created: orderBeforeCutoff,
					Items: []*pbd.OrderItem{
						{
							Id:        4003,
							ReleaseId: 5003,
							Price:     &pbd.Price{Value: 1200, Currency: "USD"},
						},
					},
				},
				{
					Id:      "order-after-cutoff",
					Status:  "Payment Received",
					Created: orderAfterCutoff,
					Items: []*pbd.OrderItem{
						{
							Id:        4004,
							ReleaseId: 5004,
							Price:     &pbd.Price{Value: 2400, Currency: "USD"},
						},
					},
				},
			},
		}

		b := background.GetBackgroundRunner(d, "", "", "")
		qc := queuelogic.GetQueue(pstore, b, di, d)

		_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: "test-subsequent-sync-lookback",
				Auth:      "123",
				Entry: &pb.QueueElement_SyncOrders{
					SyncOrders: &pb.SyncOrders{Page: 1},
				},
			},
		})
		if err != nil {
			t.Fatalf("Failed to enqueue: %v", err)
		}

		err = qc.FlushQueue(ctx)
		if err != nil {
			t.Fatalf("FlushQueue failed: %v", err)
		}

		// Item after cutoff should be synced
		saleAfter, err := d.GetSale(ctx, 123, 4004)
		if err != nil {
			t.Fatalf("Failed to get sale 4004: %v", err)
		}
		if saleAfter.GetSaleState() != pbd.SaleStatus_SOLD {
			t.Errorf("Expected sale 4004 to be SOLD, got %v", saleAfter.GetSaleState())
		}

		// Item before cutoff should NOT be synced
		_, err = d.GetSale(ctx, 123, 4003)
		if status.Code(err) != codes.NotFound {
			t.Errorf("Expected sale 4003 not found (before cutoff), got %v", err)
		}

		// User LastOrderSync should have advanced beyond syncCutoff
		u, err := d.GetUser(ctx, "123")
		if err != nil {
			t.Fatalf("Failed to get user: %v", err)
		}
		if u.GetLastOrderSync() <= syncCutoff.UnixNano() {
			t.Errorf("Expected LastOrderSync > %v, got %v", syncCutoff.UnixNano(), u.GetLastOrderSync())
		}
	})
}

func TestSyncOrders_Pagination(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	t0 := time.Now().Add(-10 * 24 * time.Hour).UnixNano()
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:          &pbd.User{DiscogsUserId: 123},
		Auth:          &pb.GramophileAuth{Token: "123"},
		LastOrderSync: t0,
	})
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Pre-create records for the 1st and 105th items to verify linking across pages
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         6001,
			InstanceId: 7001,
			FolderId:   123,
			Title:      "Page 1 Record",
		},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save record 7001: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         6105,
			InstanceId: 7105,
			FolderId:   123,
			Title:      "Page 2 Record",
		},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save record 7105: %v", err)
	}

	// Build 105 orders. TestDiscogsClient paginates at 100 per page, so this yields 2 pages.
	var orders []*pbd.Order
	orderTime := time.Now().Add(-2 * 24 * time.Hour).Unix()
	for i := int64(1); i <= 105; i++ {
		orders = append(orders, &pbd.Order{
			Id:      fmt.Sprintf("order-%d", i),
			Status:  "Payment Received",
			Created: orderTime,
			Items: []*pbd.OrderItem{
				{
					Id:        5000 + i,
					ReleaseId: 6000 + i,
					Price:     &pbd.Price{Value: int32(1000 + i), Currency: "USD"},
				},
			},
		})
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: orders,
	}

	b := background.GetBackgroundRunner(d, "", "", "")
	qc := queuelogic.GetQueue(pstore, b, di, d)

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "test-pagination",
			Auth:      "123",
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue SyncOrders: %v", err)
	}

	// Flush queue to process page 1, which enqueues page 2, which processes and enqueues LinkSales
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	// Verify all 105 sales exist and are marked SOLD
	for i := int64(1); i <= 105; i++ {
		sale, err := d.GetSale(ctx, 123, 5000+i)
		if err != nil {
			t.Fatalf("Failed to get sale %d (item %d): %v", 5000+i, i, err)
		}
		if sale.GetSaleState() != pbd.SaleStatus_SOLD {
			t.Fatalf("Expected sale %d to be SOLD, got %v", 5000+i, sale.GetSaleState())
		}
	}

	// Verify Record 7001 (page 1) linked to Sale 5001
	rec1, err := d.GetRecord(ctx, 123, 7001)
	if err != nil {
		t.Fatalf("Failed to get record 7001: %v", err)
	}
	if rec1.GetSaleId() != 5001 {
		t.Errorf("Expected record 7001 SaleId 5001, got %v", rec1.GetSaleId())
	}

	// Verify Record 7105 (page 2) linked to Sale 5105
	rec2, err := d.GetRecord(ctx, 123, 7105)
	if err != nil {
		t.Fatalf("Failed to get record 7105: %v", err)
	}
	if rec2.GetSaleId() != 5105 {
		t.Errorf("Expected record 7105 SaleId 5105, got %v", rec2.GetSaleId())
	}

	// Verify user.LastOrderSync updated to > t0
	user, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if user.GetLastOrderSync() <= t0 {
		t.Errorf("Expected user LastOrderSync > %v, got %v", t0, user.GetLastOrderSync())
	}
}

func TestSyncOrders_CancelledOrderIgnored(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:          &pbd.User{DiscogsUserId: 123},
		Auth:          &pb.GramophileAuth{Token: "123"},
		LastOrderSync: time.Now().Add(-24 * time.Hour).UnixNano(),
	})
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	// Pre-create collection record for release 9001
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         9001,
			InstanceId: 19001,
			FolderId:   123,
			Title:      "Record 9001",
		},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save record: %v", err)
	}

	// Pre-create existing sale 8001 as FOR_SALE
	err = d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       8001,
		ReleaseId:    9001,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("Failed to save initial sale: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "order-cancelled-1",
				Status:  "Cancelled",
				Created: time.Now().Unix(),
				Items: []*pbd.OrderItem{
					{
						Id:        8001,
						ReleaseId: 9001,
						Price:     &pbd.Price{Value: 1500, Currency: "USD"},
					},
				},
			},
			{
				Id:      "order-cancelled-2",
				Status:  "Cancelled (Non-Paying Buyer)",
				Created: time.Now().Unix(),
				Items: []*pbd.OrderItem{
					{
						Id:        8002,
						ReleaseId: 9002,
						Price:     &pbd.Price{Value: 1500, Currency: "USD"},
					},
				},
			},
		},
	}

	b := background.GetBackgroundRunner(d, "", "", "")
	qc := queuelogic.GetQueue(pstore, b, di, d)

	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "test-cancelled-orders",
			Auth:      "123",
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue: %v", err)
	}

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue failed: %v", err)
	}

	// Existing sale should remain FOR_SALE
	sale1, err := d.GetSale(ctx, 123, 8001)
	if err != nil {
		t.Fatalf("Failed to get sale 8001: %v", err)
	}
	if sale1.GetSaleState() != pbd.SaleStatus_FOR_SALE {
		t.Errorf("Expected cancelled order sale to remain FOR_SALE, got %v", sale1.GetSaleState())
	}
	if sale1.GetSoldDate() != 0 {
		t.Errorf("Expected SoldDate to remain 0, got %v", sale1.GetSoldDate())
	}

	// New sale 8002 from cancelled order should not have been created
	_, err = d.GetSale(ctx, 123, 8002)
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected sale 8002 not found, got %v", err)
	}
}

func TestSyncOrders_Idempotent(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	orderTime := time.Now().Add(-12 * time.Hour).Unix()
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders:       []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:          &pbd.User{DiscogsUserId: 123},
		Auth:          &pb.GramophileAuth{Token: "123"},
		LastOrderSync: time.Now().Add(-24 * time.Hour).UnixNano(),
	})
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "order-idempotent",
				Status:  "Payment Received",
				Created: orderTime,
				Items: []*pbd.OrderItem{
					{
						Id:        9501,
						ReleaseId: 9601,
						Price:     &pbd.Price{Value: 1800, Currency: "USD"},
					},
				},
			},
		},
	}

	b := background.GetBackgroundRunner(d, "", "", "")
	qc := queuelogic.GetQueue(pstore, b, di, d)

	// First run
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "test-idempotent-run-1",
			Auth:      "123",
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue run 1: %v", err)
	}
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue run 1 failed: %v", err)
	}

	sale1, err := d.GetSale(ctx, 123, 9501)
	if err != nil {
		t.Fatalf("Failed to get sale after run 1: %v", err)
	}
	if sale1.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Fatalf("Expected sale to be SOLD after run 1, got %v", sale1.GetSaleState())
	}
	updates1 := len(sale1.GetUpdates())

	// Second run with same order
	_, err = qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "test-idempotent-run-2",
			Auth:      "123",
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue run 2: %v", err)
	}
	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("FlushQueue run 2 failed: %v", err)
	}

	sale2, err := d.GetSale(ctx, 123, 9501)
	if err != nil {
		t.Fatalf("Failed to get sale after run 2: %v", err)
	}
	if sale2.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Fatalf("Expected sale to remain SOLD after run 2, got %v", sale2.GetSaleState())
	}
	if len(sale2.GetUpdates()) != updates1 {
		t.Errorf("Expected updates count not to increase on duplicate run, got %d vs %d", len(sale2.GetUpdates()), updates1)
	}
}
