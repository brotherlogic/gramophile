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
