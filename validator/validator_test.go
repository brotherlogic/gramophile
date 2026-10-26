package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	dpb "github.com/brotherlogic/discogs/proto"
	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type testQueueClient struct {
	pb.QueueServiceClient
	enqueued    []*pb.EnqueueRequest
	enqueueErr  error
	enqueueFunc func(req *pb.EnqueueRequest) (*pb.EnqueueResponse, error)
}

func (t *testQueueClient) Enqueue(ctx context.Context, req *pb.EnqueueRequest, opts ...grpc.CallOption) (*pb.EnqueueResponse, error) {
	if t.enqueueFunc != nil {
		return t.enqueueFunc(req)
	}
	if t.enqueueErr != nil {
		return nil, t.enqueueErr
	}
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

func (t *testGramophileClient) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest, opts ...grpc.CallOption) (*pb.DeleteUserResponse, error) {
	return &pb.DeleteUserResponse{}, nil
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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
		t.Errorf("expected AdjustSales to be enqueued for valid user when LastSaleAdjust > 1h even when HandlePriceUpdates is DISABLED")
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err = validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
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

	if !foundReconcile {
		t.Errorf("expected ReconcileSales to be enqueued first when both refresh and reconcile are due")
	}
	if foundRefresh {
		t.Errorf("expected RefreshSales NOT to be enqueued in the same pass when ReconcileSales is enqueued")
	}
}

func TestValidateUser_EnqueueFailsAlreadyExists_DoesNotFail(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{
		enqueueErr: status.Errorf(codes.AlreadyExists, "Already have &{page:1} in the queue"),
	}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.Add(-8 * 24 * time.Hour).UnixNano(),
		LastSaleRefresh:       now.UnixNano(),
		LastSaleReconcile:     now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("expected validateUser not to fail when enqueue returns AlreadyExists, got: %v", err)
	}
}

func TestValidateUser_EnqueueFailsGeneralError_DoesNotFail(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	queue := &testQueueClient{
		enqueueErr: fmt.Errorf("queue connection error"),
	}
	client := &testGramophileClient{}

	now := time.Now()
	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.Add(-10 * 24 * time.Hour).UnixNano(),
		LastCollectionCheck:   now.Add(-2 * time.Hour).UnixNano(),
		LastCollectionRefresh: now.Add(-48 * time.Hour).UnixNano(),
		LastSaleRefresh:       now.Add(-48 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.Add(-10 * 24 * time.Hour).UnixNano(),
		LastWantRefresh:       now.Add(-48 * time.Hour).UnixNano(),
		LastOrderSync:         now.Add(-2 * time.Hour).UnixNano(),
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				HandlePriceUpdates: pb.Enabled_ENABLED_ENABLED,
			},
		},
		LastSaleAdjust: now.Add(-2 * time.Hour).UnixNano(),
	}

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("expected validateUser not to fail on transient enqueue error, got: %v", err)
	}
}

func TestValidateUser_ReconcileSalesAlreadyExists_BlocksRefreshSales(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	refreshEnqueued := false
	queue := &testQueueClient{
		enqueueFunc: func(req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
			if req.GetElement().GetReconcileSales() != nil {
				return nil, status.Errorf(codes.AlreadyExists, "Already have ReconcileSales in queue")
			}
			if req.GetElement().GetRefreshSales() != nil {
				refreshEnqueued = true
			}
			return &pb.EnqueueResponse{}, nil
		},
	}
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

	err := validateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("expected validateUser not to fail, got: %v", err)
	}
	if refreshEnqueued {
		t.Errorf("expected RefreshSales NOT to be enqueued when ReconcileSales is already in queue")
	}
}

type failingValidatorGHClient struct {
	ghbclient.GithubridgeClient
	getIssuesErr     error
	createIssueErr   error
	getIssuesCalls   int
	createIssueCalls int
	created          []*ghbpb.CreateIssueRequest
}

func newRecordingGHClient() *failingValidatorGHClient {
	return &failingValidatorGHClient{
		GithubridgeClient: ghbclient.GetTestClient(),
	}
}

func (f *failingValidatorGHClient) GetIssues(ctx context.Context, req *ghbpb.GetIssuesRequest) (*ghbpb.GetIssuesResponse, error) {
	f.getIssuesCalls++
	if f.getIssuesErr != nil {
		return nil, f.getIssuesErr
	}
	if f.GithubridgeClient != nil {
		return f.GithubridgeClient.GetIssues(ctx, req)
	}
	return &ghbpb.GetIssuesResponse{}, nil
}

func (f *failingValidatorGHClient) CreateIssue(ctx context.Context, req *ghbpb.CreateIssueRequest) (*ghbpb.CreateIssueResponse, error) {
	f.createIssueCalls++
	if f.createIssueErr != nil {
		return nil, f.createIssueErr
	}
	f.created = append(f.created, req)
	if f.GithubridgeClient != nil {
		return f.GithubridgeClient.CreateIssue(ctx, req)
	}
	return &ghbpb.CreateIssueResponse{}, nil
}

func TestValidateUser_UserLevelStaleness_CreatesIssue(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("stale LastSaleRefresh only", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 123},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-1 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 issue to be created, got %d", len(ghClient.created))
		}
		issue := ghClient.created[0]
		if issue.GetTitle() != "Sales staleness detected for user 123" {
			t.Errorf("expected title %q, got %q", "Sales staleness detected for user 123", issue.GetTitle())
		}
		if !strings.Contains(issue.GetBody(), "LastSaleRefresh") {
			t.Errorf("expected issue body to mention LastSaleRefresh, got: %s", issue.GetBody())
		}
		if strings.Contains(issue.GetBody(), "LastSaleAdjust") {
			t.Errorf("expected issue body NOT to mention LastSaleAdjust when fresh, got: %s", issue.GetBody())
		}
	})

	t.Run("stale LastSaleAdjust only", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 456},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-1 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 issue to be created, got %d", len(ghClient.created))
		}
		issue := ghClient.created[0]
		if issue.GetTitle() != "Sales staleness detected for user 456" {
			t.Errorf("expected title %q, got %q", "Sales staleness detected for user 456", issue.GetTitle())
		}
		if !strings.Contains(issue.GetBody(), "LastSaleAdjust") {
			t.Errorf("expected issue body to mention LastSaleAdjust, got: %s", issue.GetBody())
		}
		if strings.Contains(issue.GetBody(), "LastSaleRefresh") {
			t.Errorf("expected issue body NOT to mention LastSaleRefresh when fresh, got: %s", issue.GetBody())
		}
	})

	t.Run("both LastSaleRefresh and LastSaleAdjust stale", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 789},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-9 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-10 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 consolidated issue to be created, got %d", len(ghClient.created))
		}
		issue := ghClient.created[0]
		if !strings.Contains(issue.GetBody(), "LastSaleRefresh") || !strings.Contains(issue.GetBody(), "LastSaleAdjust") {
			t.Errorf("expected issue body to mention both LastSaleRefresh and LastSaleAdjust, got: %s", issue.GetBody())
		}
	})

	t.Run("both within 7 days creates no issue", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := ghbclient.GetTestClient()

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 123},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-6 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-6 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
		if err != nil {
			t.Fatalf("failed to get issues: %v", err)
		}
		if len(resp.GetIssues()) != 0 {
			t.Fatalf("expected 0 issues when timestamps are within 7 days, got %d", len(resp.GetIssues()))
		}
	})

	t.Run("unauthenticated user with empty UserToken creates no issue", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := ghbclient.GetTestClient()

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "unauth_token"},
			UserToken:             "",
			User:                  &dpb.User{DiscogsUserId: 999},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-10 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-10 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
		if err != nil {
			t.Fatalf("failed to get issues: %v", err)
		}
		if len(resp.GetIssues()) != 0 {
			t.Fatalf("expected 0 issues for unauthenticated user (empty UserToken), got %d", len(resp.GetIssues()))
		}
	})
}

func TestValidateUser_UserLevelStaleness_Deduplicated(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)
	queue := &testQueueClient{}
	client := &testGramophileClient{}
	ghClient := ghbclient.GetTestClient()

	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "test_token"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 123},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-8 * 24 * time.Hour).UnixNano(),
		LastSaleReconcile:     now.UnixNano(),
		LastSaleAdjust:        now.Add(-8 * 24 * time.Hour).UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
	}

	// First validation pass creates the issue
	if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
		t.Fatalf("first validateUser returned unexpected error: %v", err)
	}

	// Second validation pass should deduplicate while the issue remains open
	if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
		t.Fatalf("second validateUser returned unexpected error: %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}
	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected exactly 1 issue after duplicate validation pass, got %d", len(resp.GetIssues()))
	}
}

func TestValidateUser_GithubridgeFailure_DoesNotFailValidation(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	t.Run("GetIssues error", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := &failingValidatorGHClient{
			getIssuesErr: errors.New("simulated GetIssues failure"),
		}

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 123},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("expected validateUser to return nil when GetIssues fails, got: %v", err)
		}
		if ghClient.getIssuesCalls == 0 {
			t.Errorf("expected GetIssues to be called during staleness check")
		}
	})

	t.Run("CreateIssue error", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := &failingValidatorGHClient{
			createIssueErr: errors.New("simulated CreateIssue failure"),
		}

		user := &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: "test_token"},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: 123},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("expected validateUser to return nil when CreateIssue fails, got: %v", err)
		}
		if ghClient.createIssueCalls == 0 {
			t.Errorf("expected CreateIssue to be called during staleness check")
		}
	})
}



