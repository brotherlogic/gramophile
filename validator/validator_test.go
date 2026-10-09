package main

import (
	"context"
	"testing"
	"time"

	dpb "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc"
)

type testQueueClient struct {
	pb.QueueServiceClient
	enqueued []*pb.EnqueueRequest
}

func (t *testQueueClient) Enqueue(ctx context.Context, req *pb.EnqueueRequest, opts ...grpc.CallOption) (*pb.EnqueueResponse, error) {
	t.enqueued = append(t.enqueued, req)
	return &pb.EnqueueResponse{}, nil
}

type testGramophileClient struct {
	pb.GramophileServiceClient
	users []*pb.StoredUser
}

func (t *testGramophileClient) GetUsers(ctx context.Context, req *pb.GetUsersRequest, opts ...grpc.CallOption) (*pb.GetUsersResponse, error) {
	return &pb.GetUsersResponse{Users: t.users}, nil
}

func TestAdjustSalesEnqueue_EnabledAndExpired(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				HandlePriceUpdates: pb.Enabled_ENABLED_ENABLED,
			},
		},
		LastSaleAdjust: now.Add(-2 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetAdjustSales() != nil {
			found = true
			if req.GetElement().GetIntention() != "From Validator (AdjustSales)" {
				t.Errorf("expected intention 'From Validator (AdjustSales)', got '%v'", req.GetElement().GetIntention())
			}
			if req.GetElement().GetBackoffInSeconds() != 15 {
				t.Errorf("expected backoff 15s, got %v", req.GetElement().GetBackoffInSeconds())
			}
			if req.GetElement().GetAuth() != "test_token" {
				t.Errorf("expected auth 'test_token', got '%v'", req.GetElement().GetAuth())
			}
		}
	}

	if !found {
		t.Errorf("expected AdjustSales to be enqueued when HandlePriceUpdates is ENABLED and LastSaleAdjust is > 1 hour ago")
	}
}

func TestAdjustSalesEnqueue_Disabled(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				HandlePriceUpdates: pb.Enabled_ENABLED_DISABLED,
			},
		},
		LastSaleAdjust: now.Add(-2 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetAdjustSales() != nil {
			t.Errorf("expected AdjustSales NOT to be enqueued when HandlePriceUpdates is DISABLED")
		}
	}
}

func TestAdjustSalesEnqueue_WithinOneHour(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				HandlePriceUpdates: pb.Enabled_ENABLED_ENABLED,
			},
		},
		LastSaleAdjust: now.Add(-30 * time.Minute).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetAdjustSales() != nil {
			t.Errorf("expected AdjustSales NOT to be enqueued when LastSaleAdjust is within 1 hour")
		}
	}
}

func TestSyncOrdersEnqueue_Expired(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.Add(-2 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetSyncOrders() != nil {
			found = true
			if req.GetElement().GetIntention() != "From Validator" {
				t.Errorf("expected intention 'From Validator', got '%v'", req.GetElement().GetIntention())
			}
			if req.GetElement().GetBackoffInSeconds() != 15 {
				t.Errorf("expected backoff 15s, got %v", req.GetElement().GetBackoffInSeconds())
			}
			if req.GetElement().GetAuth() != "test_token" {
				t.Errorf("expected auth 'test_token', got '%v'", req.GetElement().GetAuth())
			}
			if req.GetElement().GetSyncOrders().GetPage() != 1 {
				t.Errorf("expected Page 1, got %v", req.GetElement().GetSyncOrders().GetPage())
			}
			if req.GetElement().GetSyncOrders().GetSyncId() <= 0 {
				t.Errorf("expected SyncId > 0, got %v", req.GetElement().GetSyncOrders().GetSyncId())
			}
		}
	}

	if !found {
		t.Errorf("expected SyncOrders to be enqueued when LastOrderSync is > 1 hour ago")
	}
}

func TestSyncOrdersEnqueue_Unset(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         0,
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetSyncOrders() != nil {
			found = true
			if req.GetElement().GetSyncOrders().GetPage() != 1 {
				t.Errorf("expected Page 1, got %v", req.GetElement().GetSyncOrders().GetPage())
			}
			if req.GetElement().GetSyncOrders().GetSyncId() <= 0 {
				t.Errorf("expected SyncId > 0, got %v", req.GetElement().GetSyncOrders().GetSyncId())
			}
		}
	}

	if !found {
		t.Errorf("expected SyncOrders to be enqueued when LastOrderSync is unset (0)")
	}
}

func TestSyncOrdersEnqueue_WithinOneHour(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.Add(-30 * time.Minute).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetSyncOrders() != nil {
			t.Errorf("expected SyncOrders NOT to be enqueued when LastOrderSync is within 1 hour")
		}
	}
}

func TestReconcileSalesEnqueue_Expired(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		LastSaleReconcile:     now.Add(-8 * 24 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetReconcileSales() != nil {
			found = true
			if req.GetElement().GetIntention() != "From Validator (ReconcileSales)" {
				t.Errorf("expected intention 'From Validator (ReconcileSales)', got '%v'", req.GetElement().GetIntention())
			}
			if req.GetElement().GetBackoffInSeconds() != 15 {
				t.Errorf("expected backoff 15s, got %v", req.GetElement().GetBackoffInSeconds())
			}
			if req.GetElement().GetAuth() != "test_token" {
				t.Errorf("expected auth 'test_token', got '%v'", req.GetElement().GetAuth())
			}
			if req.GetElement().GetReconcileSales().GetPage() != 1 {
				t.Errorf("expected Page 1, got %v", req.GetElement().GetReconcileSales().GetPage())
			}
		}
	}

	if !found {
		t.Errorf("expected ReconcileSales to be enqueued when LastSaleReconcile is > 7 days ago")
	}
}

func TestReconcileSalesEnqueue_Unset(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		LastSaleReconcile:     0,
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetReconcileSales() != nil {
			found = true
			if req.GetElement().GetIntention() != "From Validator (ReconcileSales)" {
				t.Errorf("expected intention 'From Validator (ReconcileSales)', got '%v'", req.GetElement().GetIntention())
			}
			if req.GetElement().GetBackoffInSeconds() != 15 {
				t.Errorf("expected backoff 15s, got %v", req.GetElement().GetBackoffInSeconds())
			}
			if req.GetElement().GetAuth() != "test_token" {
				t.Errorf("expected auth 'test_token', got '%v'", req.GetElement().GetAuth())
			}
			if req.GetElement().GetReconcileSales().GetPage() != 1 {
				t.Errorf("expected Page 1, got %v", req.GetElement().GetReconcileSales().GetPage())
			}
		}
	}

	if !found {
		t.Errorf("expected ReconcileSales to be enqueued when LastSaleReconcile is unset (0)")
	}
}

func TestReconcileSalesEnqueue_WithinSevenDays(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		LastSaleReconcile:     now.Add(-24 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetReconcileSales() != nil {
			t.Errorf("expected ReconcileSales NOT to be enqueued when LastSaleReconcile is within 7 days")
		}
	}
}

func TestSalesSync_SkippedWhenActiveSyncInFlight(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-25 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.Add(-8 * 24 * time.Hour).UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		SaleSyncActiveTime:    now.Add(-1 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshSales() != nil {
			t.Errorf("expected RefreshSales NOT to be enqueued when SaleSyncActiveTime is within 24h")
		}
		if req.GetElement().GetReconcileSales() != nil {
			t.Errorf("expected ReconcileSales NOT to be enqueued when SaleSyncActiveTime is within 24h")
		}
	}
}

func TestSalesSync_EnqueuedWhenActiveSyncStale(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-25 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		SaleSyncActiveTime:    now.Add(-25 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshSales() != nil {
			found = true
		}
	}
	if !found {
		t.Errorf("expected RefreshSales to be enqueued when SaleSyncActiveTime is older than 24h")
	}
}

func TestSalesSync_SetsSaleSyncActiveTimeOnEnqueue(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-25 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		SaleSyncActiveTime:    0,
	}

	err := tdb.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("unable to save user: %v", err)
	}

	err = validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	savedUser, err := tdb.GetUser(ctx, user.GetAuth().GetToken())
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}

	if savedUser.GetSaleSyncActiveTime() == 0 {
		t.Errorf("expected SaleSyncActiveTime to be set in DB, got 0")
	} else if time.Since(time.Unix(0, savedUser.GetSaleSyncActiveTime())) > time.Minute {
		t.Errorf("expected SaleSyncActiveTime to be set to current time, got %v", savedUser.GetSaleSyncActiveTime())
	}
}

func TestSalesSync_ConcurrentMutualExclusion(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-25 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.Add(-8 * 24 * time.Hour).UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		SaleSyncActiveTime:    0,
	}

	err := validateUser(ctx, user, client, queue, tdb)
	if err != nil {
		t.Fatalf("validateUser returned unexpected error: %v", err)
	}

	foundRefresh := false
	foundReconcile := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshSales() != nil {
			foundRefresh = true
		}
		if req.GetElement().GetReconcileSales() != nil {
			foundReconcile = true
		}
	}

	if !foundRefresh {
		t.Errorf("expected RefreshSales to be enqueued when both refresh and reconcile are due")
	}
	if foundReconcile {
		t.Errorf("expected ReconcileSales NOT to be enqueued in the same pass when RefreshSales is enqueued")
	}
}

