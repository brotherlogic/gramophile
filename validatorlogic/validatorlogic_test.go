package validatorlogic

import (
	"context"
	"testing"
	"time"

	dpb "github.com/brotherlogic/discogs/proto"
	ghbclient "github.com/brotherlogic/githubridge/client"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc"
)

type testQueueClient struct {
	pb.QueueServiceClient
	enqueued   []*pb.EnqueueRequest
	enqueueErr error
}

func (t *testQueueClient) Enqueue(ctx context.Context, req *pb.EnqueueRequest, opts ...grpc.CallOption) (*pb.EnqueueResponse, error) {
	if t.enqueueErr != nil {
		return nil, t.enqueueErr
	}
	t.enqueued = append(t.enqueued, req)
	return &pb.EnqueueResponse{}, nil
}

type testGramophileClient struct {
	pb.GramophileServiceClient
}

func (t *testGramophileClient) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest, opts ...grpc.CallOption) (*pb.DeleteUserResponse, error) {
	return &pb.DeleteUserResponse{}, nil
}

func TestValidateUser_RefreshWantlists_Stale(t *testing.T) {
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
		LastSaleReconcile:     now.UnixNano(),
		LastSaleAdjust:        now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastWantlistRefresh:   now.Add(-25 * time.Hour).UnixNano(),
		LastOrderSync:         now.UnixNano(),
	}

	err := ValidateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("ValidateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshWantlists() != nil {
			found = true
			if req.GetElement().GetIntention() != "From Validator" {
				t.Errorf("expected intention 'From Validator', got %q", req.GetElement().GetIntention())
			}
			if req.GetElement().GetBackoffInSeconds() != 15 {
				t.Errorf("expected backoff 15s, got %v", req.GetElement().GetBackoffInSeconds())
			}
			if req.GetElement().GetAuth() != "test_token" {
				t.Errorf("expected auth 'test_token', got %q", req.GetElement().GetAuth())
			}
		}
	}

	if !found {
		t.Errorf("expected RefreshWantlists to be enqueued when LastWantlistRefresh is > 24 hours ago")
	}
}

func TestValidateUser_RefreshWantlists_Unset(t *testing.T) {
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
		LastSaleReconcile:     now.UnixNano(),
		LastSaleAdjust:        now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastWantlistRefresh:   0,
		LastOrderSync:         now.UnixNano(),
	}

	err := ValidateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("ValidateUser returned unexpected error: %v", err)
	}

	found := false
	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshWantlists() != nil {
			found = true
		}
	}

	if !found {
		t.Errorf("expected RefreshWantlists to be enqueued when LastWantlistRefresh is unset (0)")
	}
}

func TestValidateUser_RefreshWantlists_Fresh(t *testing.T) {
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
		LastSaleReconcile:     now.UnixNano(),
		LastSaleAdjust:        now.UnixNano(),
		LastWantRefresh:       now.UnixNano(),
		LastWantlistRefresh:   now.Add(-1 * time.Hour).UnixNano(),
		LastOrderSync:         now.UnixNano(),
	}

	err := ValidateUser(ctx, user, client, queue, tdb, ghbclient.GetTestClient())
	if err != nil {
		t.Fatalf("ValidateUser returned unexpected error: %v", err)
	}

	for _, req := range queue.enqueued {
		if req.GetElement().GetRefreshWantlists() != nil {
			t.Errorf("expected RefreshWantlists NOT to be enqueued when LastWantlistRefresh is within 24 hours")
		}
	}
}

func TestShouldSaleHaveUpdatedWithinWeek_MissingMetadataExcluded(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name       string
		sale       *pb.SaleInfo
		config     *pb.SaleConfig
		wantUpdate bool
	}{
		{
			name: "REDUCE_TO_MEDIAN with MedianPrice == 0 is excluded",
			sale: &pb.SaleInfo{
				SaleId:          1,
				ReleaseId:       100,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2000},
				MedianPrice:     &dpb.Price{Value: 0},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              100,
			},
			wantUpdate: false,
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW with MedianPrice == 0 is excluded",
			sale: &pb.SaleInfo{
				SaleId:          2,
				ReleaseId:       200,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2000},
				MedianPrice:     &dpb.Price{Value: 0},
				LowPrice:        &dpb.Price{Value: 500},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:              100,
			},
			wantUpdate: false,
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE with MedianPrice == 0 is excluded",
			sale: &pb.SaleInfo{
				SaleId:          3,
				ReleaseId:       300,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2000},
				MedianPrice:     &dpb.Price{Value: 0},
				LowPrice:        &dpb.Price{Value: 500},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:     pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds: 3600,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE,
				Reduction:              100,
			},
			wantUpdate: false,
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW at/post-median with DISCOGS_LOW and LowPrice == 0 is excluded",
			sale: &pb.SaleInfo{
				SaleId:          4,
				ReleaseId:       400,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 1500},
				MedianPrice:     &dpb.Price{Value: 1500},
				LowPrice:        &dpb.Price{Value: 0},
				TimeAtMedian:    now.Add(-10 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       3600,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				PostMedianTime:               3600,
				PostMedianReduction:          100,
				PostMedianReductionFrequency: 3600,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
			},
			wantUpdate: false,
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE at/post-median with DISCOGS_LOW and LowPrice == 0 is excluded",
			sale: &pb.SaleInfo{
				SaleId:          5,
				ReleaseId:       500,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 1500},
				MedianPrice:     &dpb.Price{Value: 1500},
				LowPrice:        &dpb.Price{Value: 0},
				TimeAtMedian:    now.Add(-10 * 24 * time.Hour).UnixNano(),
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       3600,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE,
				PostMedianTime:               3600,
				PostMedianReduction:          100,
				PostMedianReductionFrequency: 3600,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
				StaleBound:                   500,
			},
			wantUpdate: false,
		},
		{
			name: "REDUCE_TO_MEDIAN_AND_THEN_LOW pre-median with MedianPrice > 0 and LowPrice == 0 still expects update",
			sale: &pb.SaleInfo{
				SaleId:          6,
				ReleaseId:       600,
				SaleState:       dpb.SaleStatus_FOR_SALE,
				CurrentPrice:    &dpb.Price{Value: 2500},
				MedianPrice:     &dpb.Price{Value: 1500},
				LowPrice:        &dpb.Price{Value: 0},
				LastPriceUpdate: now.Add(-10 * 24 * time.Hour).UnixNano(),
			},
			config: &pb.SaleConfig{
				HandlePriceUpdates:           pb.Enabled_ENABLED_ENABLED,
				UpdateFrequencySeconds:       3600,
				UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
				Reduction:                    100,
				PostMedianTime:               3600,
				PostMedianReduction:          100,
				PostMedianReductionFrequency: 3600,
				LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
			},
			wantUpdate: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, _ := shouldSaleHaveUpdatedWithinWeek(tc.sale, tc.config)
			if got != tc.wantUpdate {
				t.Errorf("shouldSaleHaveUpdatedWithinWeek() = %v, want %v", got, tc.wantUpdate)
			}
		})
	}
}
