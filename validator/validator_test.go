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

func TestValidateUser_PerSaleStaleness_CreatesIssue(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	newFreshUserWithUpdatesEnabled := func(userID int32, cfg *pb.SaleConfig) *pb.StoredUser {
		return &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: fmt.Sprintf("token_%d", userID)},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: userID},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-1 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-30 * time.Minute).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
			Config: &pb.GramophileConfig{
				SaleConfig: cfg,
			},
		}
	}

	t.Run("active FOR_SALE listing above median with LastPriceUpdate > 7d", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := newFreshUserWithUpdatesEnabled(101, &pb.SaleConfig{
			HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
			UpdateFrequencySeconds: 3600,
			UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			Reduction:              100,
		})
		if err := tdb.SaveSale(ctx, 101, &pb.SaleInfo{
			SaleId:          9001,
			ReleaseId:       8001,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
			MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			LastPriceUpdate: now.Add(-8 * 24 * time.Hour).UnixNano(),
		}); err != nil {
			t.Fatalf("failed to save sale: %v", err)
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 issue to be created, got %d", len(ghClient.created))
		}
		body := ghClient.created[0].GetBody()
		if !strings.Contains(body, "9001") {
			t.Errorf("expected issue body to include SaleId 9001, got: %s", body)
		}
		if !strings.Contains(body, "8001") {
			t.Errorf("expected issue body to include ReleaseId 8001, got: %s", body)
		}
		if !strings.Contains(body, "REDUCE_TO_MEDIAN") {
			t.Errorf("expected issue body to include UpdateType REDUCE_TO_MEDIAN, got: %s", body)
		}
	})

	t.Run("active FOR_SALE listing stuck on missing MedianPrice == 0 or missing condition (EC4)", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := newFreshUserWithUpdatesEnabled(102, &pb.SaleConfig{
			HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
			UpdateFrequencySeconds: 3600,
			UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			Reduction:              100,
		})
		// Sale 9002 stuck on MedianPrice == 0
		if err := tdb.SaveSale(ctx, 102, &pb.SaleInfo{
			SaleId:          9002,
			ReleaseId:       8002,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
			LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
		}); err != nil {
			t.Fatalf("failed to save sale 9002: %v", err)
		}
		// Sale 9003 stuck on missing condition
		if err := tdb.SaveSale(ctx, 102, &pb.SaleInfo{
			SaleId:          9003,
			ReleaseId:       8003,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
			MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
			Condition:       "",
			SleeveCondition: "",
			LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
		}); err != nil {
			t.Fatalf("failed to save sale 9003: %v", err)
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 issue to be created for EC4 stuck sales, got %d", len(ghClient.created))
		}
		body := ghClient.created[0].GetBody()
		if !strings.Contains(body, "9002") || !strings.Contains(body, "9003") {
			t.Errorf("expected issue body to include both stuck SaleIds 9002 and 9003, got: %s", body)
		}
	})

	t.Run("active FOR_SALE listing with SaleUpdateOverride enabling updates", func(t *testing.T) {
		pstore := pstore_client.GetTestClient()
		tdb := db.NewTestDB(pstore)
		queue := &testQueueClient{}
		client := &testGramophileClient{}
		ghClient := newRecordingGHClient()

		user := newFreshUserWithUpdatesEnabled(103, &pb.SaleConfig{
			HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
			UpdateFrequencySeconds: 3600,
			UpdateType:             pb.SaleUpdateType_NO_SALE_UPDATE,
		})
		if err := tdb.SaveSale(ctx, 103, &pb.SaleInfo{
			SaleId:             9004,
			ReleaseId:          8004,
			SaleState:          dpb.SaleStatus_FOR_SALE,
			CurrentPrice:       &dpb.Price{Value: 1500, Currency: "USD"},
			SaleUpdateOverride: pb.SaleUpdateType_MINIMAL_REDUCE,
			LastPriceUpdate:    now.Add(-8 * 24 * time.Hour).UnixNano(),
		}); err != nil {
			t.Fatalf("failed to save sale 9004: %v", err)
		}

		if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
			t.Fatalf("validateUser returned unexpected error: %v", err)
		}

		if len(ghClient.created) != 1 {
			t.Fatalf("expected 1 issue to be created, got %d", len(ghClient.created))
		}
		if !strings.Contains(ghClient.created[0].GetBody(), "9004") {
			t.Errorf("expected issue body to include SaleId 9004, got: %s", ghClient.created[0].GetBody())
		}
	})
}

func TestValidateUser_PerSaleHoldingStates_NoIssue(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	newUser := func(userID int32, cfg *pb.SaleConfig) *pb.StoredUser {
		return &pb.StoredUser{
			Auth:                  &pb.GramophileAuth{Token: fmt.Sprintf("token_%d", userID)},
			UserToken:             "user_token",
			User:                  &dpb.User{DiscogsUserId: userID},
			LastRefreshTime:       now.UnixNano(),
			LastCollectionCheck:   now.UnixNano(),
			LastCollectionRefresh: now.UnixNano(),
			LastSaleRefresh:       now.Add(-1 * time.Hour).UnixNano(),
			LastSaleReconcile:     now.UnixNano(),
			LastSaleAdjust:        now.Add(-30 * time.Minute).UnixNano(),
			LastWantRefresh:       now.UnixNano(),
			LastOrderSync:         now.UnixNano(),
			Config: &pb.GramophileConfig{
				SaleConfig: cfg,
			},
		}
	}

	cases := []struct {
		name string
		cfg  *pb.SaleConfig
		sale *pb.SaleInfo
	}{
		{
			name: "HandlePriceUpdates disabled",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_DISABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			},
			sale: &pb.SaleInfo{
				SaleId:          7001,
				ReleaseId:       6001,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "NO_SALE_UPDATE configured",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_NO_SALE_UPDATE,
			},
			sale: &pb.SaleInfo{
				SaleId:          7002,
				ReleaseId:       6002,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "SOLD listing",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			},
			sale: &pb.SaleInfo{
				SaleId:          7003,
				ReleaseId:       6003,
				SaleState:       dpb.SaleStatus_SOLD,
				CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN holding with TimeAtMedian > 0",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			},
			sale: &pb.SaleInfo{
				SaleId:          7004,
				ReleaseId:       6004,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 1200, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				TimeAtMedian:    now.Add(-9 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN holding with CurrentPrice <= MedianPrice",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
			},
			sale: &pb.SaleInfo{
				SaleId:          7005,
				ReleaseId:       6005,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 1100, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW within PostMedianTime holding window",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       3600,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				PostMedianTime:               int64((10 * 24 * time.Hour).Seconds()),
				PostMedianReduction:          100,
				PostMedianReductionFrequency: 86400,
				LowerBound:                   800,
			},
			sale: &pb.SaleInfo{
				SaleId:          7006,
				ReleaseId:       6006,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 1200, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				TimeAtMedian:    now.Add(-8 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-8 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW at lower bound",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       3600,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				PostMedianTime:               3600,
				PostMedianReduction:          100,
				PostMedianReductionFrequency: 3600,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
			},
			sale: &pb.SaleInfo{
				SaleId:          7007,
				ReleaseId:       6007,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 900, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				LowPrice:        &dpb.Price{Value: 900, Currency: "USD"},
				TimeAtMedian:    now.Add(-15 * 24 * time.Hour).UnixNano(),
				TimeAtLow:       now.Add(-9 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE within TimeToStaleDays holding window",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:                pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:            3600,
				UpdateType:                        pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE,
				PostMedianTime:                    3600,
				PostMedianReduction:               100,
				PostMedianReductionFrequency:      3600,
				LowerBound:                        900,
				TimeToStaleDays:                   14,
				PostLowReduction:                  50,
				PostLowReductionFrequencySeconds:  86400,
				StaleBound:                        500,
			},
			sale: &pb.SaleInfo{
				SaleId:          7008,
				ReleaseId:       6008,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 900, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				TimeAtMedian:    now.Add(-20 * 24 * time.Hour).UnixNano(),
				TimeAtLow:       now.Add(-10 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE at stale bound",
			cfg: &pb.SaleConfig{
				HandlePriceUpdates:                pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:            3600,
				UpdateType:                        pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE,
				PostMedianTime:                    3600,
				PostMedianReduction:               100,
				PostMedianReductionFrequency:      3600,
				LowerBound:                        900,
				TimeToStaleDays:                   5,
				PostLowReduction:                  50,
				PostLowReductionFrequencySeconds:  3600,
				StaleBound:                        500,
			},
			sale: &pb.SaleInfo{
				SaleId:          7009,
				ReleaseId:       6009,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 500, Currency: "USD"},
				MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
				TimeAtMedian:    now.Add(-30 * 24 * time.Hour).UnixNano(),
				TimeAtLow:       now.Add(-20 * 24 * time.Hour).UnixNano(),
				TimeAtStale:     now.Add(-9 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
			},
		},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pstore := pstore_client.GetTestClient()
			tdb := db.NewTestDB(pstore)
			queue := &testQueueClient{}
			client := &testGramophileClient{}
			ghClient := newRecordingGHClient()

			userID := int32(200 + i)
			user := newUser(userID, tc.cfg)
			if err := tdb.SaveSale(ctx, userID, tc.sale); err != nil {
				t.Fatalf("failed to save sale: %v", err)
			}

			if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
				t.Fatalf("validateUser returned unexpected error: %v", err)
			}

			if len(ghClient.created) != 0 {
				t.Fatalf("expected 0 issues for %s, got %d: %v", tc.name, len(ghClient.created), ghClient.created[0].GetBody())
			}
		})
	}
}

func TestValidateUser_StalenessIssue_ConsolidatedAndDeduplicated(t *testing.T) {
	ctx := context.Background()
	now := time.Now()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)
	queue := &testQueueClient{}
	client := &testGramophileClient{}
	ghClient := newRecordingGHClient()

	user := &pb.StoredUser{
		Auth:                  &pb.GramophileAuth{Token: "token_333"},
		UserToken:             "user_token",
		User:                  &dpb.User{DiscogsUserId: 333},
		LastRefreshTime:       now.UnixNano(),
		LastCollectionCheck:   now.UnixNano(),
		LastCollectionRefresh: now.UnixNano(),
		LastSaleRefresh:       now.Add(-9 * 24 * time.Hour).UnixNano(), // user-level stale
		LastSaleReconcile:     now.UnixNano(),
		LastSaleAdjust:        now.Add(-1 * time.Hour).UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastOrderSync:         now.UnixNano(),
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              100,
			},
		},
	}
	if err := tdb.SaveUser(ctx, user); err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// Two active stale sales (5001, 5002) and one holding sale (5003)
	sales := []*pb.SaleInfo{
		{
			SaleId:          5001,
			ReleaseId:       4001,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 2000, Currency: "USD"},
			MedianPrice:     &dpb.Price{Value: 1200, Currency: "USD"},
			LastPriceUpdate: now.Add(-8 * 24 * time.Hour).UnixNano(),
		},
		{
			SaleId:          5002,
			ReleaseId:       4002,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 1800, Currency: "USD"},
			MedianPrice:     &dpb.Price{Value: 1000, Currency: "USD"},
			LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
		},
		{
			SaleId:          5003,
			ReleaseId:       4003,
			SaleState:       dpb.SaleStatus_FOR_SALE,
			CurrentPrice:    &dpb.Price{Value: 1000, Currency: "USD"},
			MedianPrice:     &dpb.Price{Value: 1000, Currency: "USD"},
			TimeAtMedian:    now.Add(-8 * 24 * time.Hour).UnixNano(),
			LastPriceUpdate: now.Add(-9 * 24 * time.Hour).UnixNano(),
		},
	}
	for _, s := range sales {
		if err := tdb.SaveSale(ctx, 333, s); err != nil {
			t.Fatalf("failed to save sale %v: %v", s.GetSaleId(), err)
		}
	}

	// Pass 1: creates a single consolidated issue containing user-level and per-sale staleness
	if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
		t.Fatalf("first validateUser returned unexpected error: %v", err)
	}
	if len(ghClient.created) != 1 {
		t.Fatalf("expected 1 consolidated issue created, got %d", len(ghClient.created))
	}
	body := ghClient.created[0].GetBody()
	if !strings.Contains(body, "LastSaleRefresh") {
		t.Errorf("expected consolidated issue body to mention LastSaleRefresh, got: %s", body)
	}
	if !strings.Contains(body, "5001") || !strings.Contains(body, "5002") {
		t.Errorf("expected consolidated issue body to mention stale sales 5001 and 5002, got: %s", body)
	}
	if strings.Contains(body, "5003") {
		t.Errorf("expected consolidated issue body NOT to mention holding sale 5003, got: %s", body)
	}

	// Pass 2: skips creating a duplicate while the issue remains open
	if err := validateUser(ctx, user, client, queue, tdb, ghClient); err != nil {
		t.Fatalf("second validateUser returned unexpected error: %v", err)
	}
	if len(ghClient.created) != 1 {
		t.Fatalf("expected issue creation to be deduplicated on second pass, got %d created issues", len(ghClient.created))
	}
}




