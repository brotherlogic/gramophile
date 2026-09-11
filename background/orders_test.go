package background

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

type capturingDiscogsClient struct {
	*discogs.TestDiscogsClient
	capturedCreatedAfter time.Time
	capturedPage         int32
	pages                int32
	ordersMap            map[int32][]*pbd.Order
}

func (c *capturingDiscogsClient) ListOrders(ctx context.Context, createdAfter time.Time, page int32) ([]*pbd.Order, *pbd.Pagination, error) {
	c.capturedCreatedAfter = createdAfter
	c.capturedPage = page
	if c.ordersMap != nil {
		orders := c.ordersMap[page]
		return orders, &pbd.Pagination{Page: page, Pages: c.pages}, nil
	}
	orders, pagination, err := c.TestDiscogsClient.ListOrders(ctx, createdAfter, page)
	if c.pages > 0 {
		pagination.Pages = c.pages
		pagination.Page = page
	}
	return orders, pagination, err
}

func TestSyncOrders_TransitionExistingSaleToSold(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	orderCreated := time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC).Unix()

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "150295-1",
				Status:  "Payment Received",
				Created: orderCreated,
				Items: []*pbd.OrderItem{
					{
						Id:        12345,
						ReleaseId: 8419904,
						Price:     &pbd.Price{Value: 1200, Currency: "USD"},
						Condition: "Very Good Plus (VG+)",
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       12345,
		ReleaseId:    8419904,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
		Condition:    "Very Good Plus (VG+)",
	})
	if err != nil {
		t.Fatalf("Failed to save initial sale: %v", err)
	}

	b := GetBackgroundRunner(d, "", "", "")
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	}

	pagination, err := b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}
	if pagination == nil {
		t.Fatalf("Expected non-nil pagination")
	}

	sale, err := d.GetSale(ctx, 123, 12345)
	if err != nil {
		t.Fatalf("Failed to get sale: %v", err)
	}

	if sale.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected SaleState to be SOLD, got %v", sale.GetSaleState())
	}
	expectedSoldDate := time.Unix(orderCreated, 0).UnixNano()
	if sale.GetSoldDate() != expectedSoldDate {
		t.Errorf("Expected SoldDate %v, got %v", expectedSoldDate, sale.GetSoldDate())
	}
	if sale.GetCurrentPrice().GetValue() != 1200 {
		t.Errorf("Expected CurrentPrice 1200, got %v", sale.GetCurrentPrice().GetValue())
	}
	if len(sale.GetUpdates()) == 0 {
		t.Errorf("Expected price update to be recorded since price changed")
	}
}

func TestSyncOrders_CreateNewSoldSale(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	orderCreated := time.Date(2023, 8, 20, 10, 0, 0, 0, time.UTC).Unix()

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "150295-2",
				Status:  "Shipped",
				Created: orderCreated,
				Items: []*pbd.OrderItem{
					{
						Id:        54321,
						ReleaseId: 9999,
						Price:     &pbd.Price{Value: 2500, Currency: "USD"},
						Condition: "Mint (M)",
					},
				},
			},
		},
	}

	ctx := context.Background()
	b := GetBackgroundRunner(d, "", "", "")
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: time.Date(2023, 8, 1, 0, 0, 0, 0, time.UTC).UnixNano(),
	}

	_, err := b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}

	sale, err := d.GetSale(ctx, 123, 54321)
	if err != nil {
		t.Fatalf("Failed to retrieve created sale: %v", err)
	}

	if sale.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected SaleState SOLD, got %v", sale.GetSaleState())
	}
	expectedSoldDate := time.Unix(orderCreated, 0).UnixNano()
	if sale.GetSoldDate() != expectedSoldDate {
		t.Errorf("Expected SoldDate %v, got %v", expectedSoldDate, sale.GetSoldDate())
	}
	if sale.GetCurrentPrice().GetValue() != 2500 {
		t.Errorf("Expected CurrentPrice 2500, got %v", sale.GetCurrentPrice().GetValue())
	}
}

func TestSyncOrders_CancelledOrderIgnored(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "150295-3",
				Status:  "Cancelled",
				Created: time.Now().Unix(),
				Items: []*pbd.OrderItem{
					{
						Id:        67890,
						ReleaseId: 1111,
						Price:     &pbd.Price{Value: 1500, Currency: "USD"},
					},
				},
			},
			{
				Id:      "150295-4",
				Status:  "Cancelled (Non-Paying Buyer)",
				Created: time.Now().Unix(),
				Items: []*pbd.OrderItem{
					{
						Id:        67891,
						ReleaseId: 1112,
						Price:     &pbd.Price{Value: 1500, Currency: "USD"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	err := d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       67890,
		ReleaseId:    1111,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		CurrentPrice: &pbd.Price{Value: 1500, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("Failed to save initial sale: %v", err)
	}

	b := GetBackgroundRunner(d, "", "", "")
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: time.Now().Add(-24 * time.Hour).UnixNano(),
	}

	_, err = b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}

	sale, err := d.GetSale(ctx, 123, 67890)
	if err != nil {
		t.Fatalf("Failed to get sale: %v", err)
	}
	if sale.GetSaleState() != pbd.SaleStatus_FOR_SALE {
		t.Errorf("Expected cancelled order item to remain FOR_SALE, got %v", sale.GetSaleState())
	}
	if sale.GetSoldDate() != 0 {
		t.Errorf("Expected SoldDate to remain 0, got %v", sale.GetSoldDate())
	}

	// 67891 should not have been created
	_, err = d.GetSale(ctx, 123, 67891)
	if err == nil {
		t.Errorf("Expected sale 67891 from cancelled order not to exist")
	}
}

func TestSyncOrders_MultiItemOrder(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	orderCreated := time.Now().Add(-12 * time.Hour).Unix()

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "150295-multi",
				Status:  "Payment Received",
				Created: orderCreated,
				Items: []*pbd.OrderItem{
					{
						Id:        101,
						ReleaseId: 201,
						Price:     &pbd.Price{Value: 1000, Currency: "USD"},
					},
					{
						Id:        102,
						ReleaseId: 202,
						Price:     &pbd.Price{Value: 2000, Currency: "USD"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	// Item 101 exists in DB as FOR_SALE
	err := d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:       101,
		ReleaseId:    201,
		SaleState:    pbd.SaleStatus_FOR_SALE,
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
	})
	if err != nil {
		t.Fatalf("Failed to save sale: %v", err)
	}

	b := GetBackgroundRunner(d, "", "", "")
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: time.Now().Add(-24 * time.Hour).UnixNano(),
	}

	_, err = b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}

	s1, err := d.GetSale(ctx, 123, 101)
	if err != nil || s1.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Sale 101 was not properly updated to SOLD: %v, err: %v", s1, err)
	}

	s2, err := d.GetSale(ctx, 123, 102)
	if err != nil || s2.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Sale 102 was not properly created as SOLD: %v, err: %v", s2, err)
	}
}

func TestSyncOrders_ColdStartLookback(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	baseClient := &discogs.TestDiscogsClient{UserId: 123}
	di := &capturingDiscogsClient{TestDiscogsClient: baseClient}

	ctx := context.Background()
	b := GetBackgroundRunner(d, "", "", "")

	// Cold start user: LastOrderSync is 0
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: 0,
	}

	start := time.Now()
	_, err := b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}

	expectedLookback := start.Add(-30 * 24 * time.Hour)
	diff := di.capturedCreatedAfter.Sub(expectedLookback)
	if diff < -time.Minute || diff > time.Minute {
		t.Errorf("Expected cold start lookback around 30 days ago (%v), got %v", expectedLookback, di.capturedCreatedAfter)
	}

	// Non-cold start user
	lastSync := time.Date(2023, 8, 15, 12, 0, 0, 0, time.UTC)
	user.LastOrderSync = lastSync.UnixNano()

	_, err = b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("SyncOrders failed: %v", err)
	}

	if !di.capturedCreatedAfter.Equal(lastSync) {
		t.Errorf("Expected incremental lookback to equal LastOrderSync (%v), got %v", lastSync, di.capturedCreatedAfter)
	}
}

func TestSyncOrders_Idempotent(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	orderCreated := time.Now().Add(-12 * time.Hour).Unix()

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Orders: []*pbd.Order{
			{
				Id:      "150295-idempotent",
				Status:  "Payment Received",
				Created: orderCreated,
				Items: []*pbd.OrderItem{
					{
						Id:        909,
						ReleaseId: 808,
						Price:     &pbd.Price{Value: 1000, Currency: "USD"},
					},
				},
			},
		},
	}

	ctx := context.Background()
	b := GetBackgroundRunner(d, "", "", "")
	user := &pb.StoredUser{
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: time.Now().Add(-24 * time.Hour).UnixNano(),
	}

	// First execution
	_, err := b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("First SyncOrders failed: %v", err)
	}

	sale1, err := d.GetSale(ctx, 123, 909)
	if err != nil {
		t.Fatalf("Failed to get sale: %v", err)
	}
	updatesCount1 := len(sale1.GetUpdates())

	// Second execution (same order)
	_, err = b.SyncOrders(ctx, di, user, 1)
	if err != nil {
		t.Fatalf("Second SyncOrders failed: %v", err)
	}

	sale2, err := d.GetSale(ctx, 123, 909)
	if err != nil {
		t.Fatalf("Failed to get sale after second run: %v", err)
	}

	if sale2.GetSaleState() != pbd.SaleStatus_SOLD {
		t.Errorf("Expected sale to still be SOLD, got %v", sale2.GetSaleState())
	}
	if len(sale2.GetUpdates()) != updatesCount1 {
		t.Errorf("Expected updates count not to grow on duplicate run, got %v vs %v", len(sale2.GetUpdates()), updatesCount1)
	}
}

func TestProcessSyncOrders_Pagination(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	baseClient := &discogs.TestDiscogsClient{UserId: 123}
	di := &capturingDiscogsClient{
		TestDiscogsClient: baseClient,
		pages:             3,
	}

	ctx := context.Background()
	b := GetBackgroundRunner(d, "", "", "")

	user := &pb.StoredUser{
		Auth:          &pb.GramophileAuth{Token: "123"},
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: 100,
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	entry := &pb.QueueElement{
		Entry: &pb.QueueElement_SyncOrders{
			SyncOrders: &pb.SyncOrders{
				Page:   1,
				SyncId: 99999,
			},
		},
	}

	var enqueuedRequests []*pb.EnqueueRequest
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueuedRequests = append(enqueuedRequests, req)
		return &pb.EnqueueResponse{}, nil
	}

	err = b.ProcessSyncOrders(ctx, di, user, entry, enqueue)
	if err != nil {
		t.Fatalf("ProcessSyncOrders failed: %v", err)
	}

	// Should enqueue page 2 with same SyncId
	if len(enqueuedRequests) != 1 {
		t.Fatalf("Expected 1 enqueued request for next page, got %d", len(enqueuedRequests))
	}

	nextElem := enqueuedRequests[0].GetElement()
	syncOrdersElem := nextElem.GetSyncOrders()
	if syncOrdersElem == nil {
		t.Fatalf("Expected SyncOrders element enqueued")
	}
	if syncOrdersElem.GetPage() != 2 {
		t.Errorf("Expected enqueued page 2, got %v", syncOrdersElem.GetPage())
	}
	if syncOrdersElem.GetSyncId() != 99999 {
		t.Errorf("Expected enqueued SyncId 99999, got %v", syncOrdersElem.GetSyncId())
	}

	// User LastOrderSync should not have been updated yet
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if u.GetLastOrderSync() != 100 {
		t.Errorf("Expected LastOrderSync to still be 100, got %v", u.GetLastOrderSync())
	}
}

func TestProcessSyncOrders_FinalPageLinkSales(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	baseClient := &discogs.TestDiscogsClient{UserId: 123}
	di := &capturingDiscogsClient{
		TestDiscogsClient: baseClient,
		pages:             2,
	}

	ctx := context.Background()
	b := GetBackgroundRunner(d, "", "", "")

	user := &pb.StoredUser{
		Auth:          &pb.GramophileAuth{Token: "123"},
		User:          &pbd.User{DiscogsUserId: 123},
		LastOrderSync: 100,
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	entry := &pb.QueueElement{
		Entry: &pb.QueueElement_SyncOrders{
			SyncOrders: &pb.SyncOrders{
				Page:   2,
				SyncId: 88888,
			},
		},
	}

	var enqueuedRequests []*pb.EnqueueRequest
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueuedRequests = append(enqueuedRequests, req)
		return &pb.EnqueueResponse{}, nil
	}

	startTime := time.Now().UnixNano()
	err = b.ProcessSyncOrders(ctx, di, user, entry, enqueue)
	if err != nil {
		t.Fatalf("ProcessSyncOrders failed: %v", err)
	}

	// Final page should update user.LastOrderSync and enqueue LinkSales
	u, err := d.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("Failed to get user: %v", err)
	}
	if u.GetLastOrderSync() < startTime {
		t.Errorf("Expected LastOrderSync to be updated to >= %v, got %v", startTime, u.GetLastOrderSync())
	}

	if len(enqueuedRequests) != 1 {
		t.Fatalf("Expected 1 enqueued request for LinkSales, got %d", len(enqueuedRequests))
	}

	nextElem := enqueuedRequests[0].GetElement()
	linkSalesElem := nextElem.GetLinkSales()
	if linkSalesElem == nil {
		t.Fatalf("Expected LinkSales element enqueued")
	}
	if linkSalesElem.GetRefreshId() != 88888 {
		t.Errorf("Expected LinkSales RefreshId 88888, got %v", linkSalesElem.GetRefreshId())
	}
}
