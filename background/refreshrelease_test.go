package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type statsTestDiscogsClient struct {
	*discogs.TestDiscogsClient
	statsResponse map[int64]*pbd.ReleaseStats
	statsError    map[int64]error
}

func (s *statsTestDiscogsClient) GetReleaseStats(ctx context.Context, releaseId int64) (*pbd.ReleaseStats, error) {
	if err, exists := s.statsError[releaseId]; exists && err != nil {
		return nil, err
	}
	if st, exists := s.statsResponse[releaseId]; exists {
		return st, nil
	}
	return s.TestDiscogsClient.GetReleaseStats(ctx, releaseId)
}

func (s *statsTestDiscogsClient) ForUser(user *pbd.User) discogs.Discogs {
	s.TestDiscogsClient.ForUser(user)
	return s
}

func TestRefreshRelease_StatsSuccess(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	baseClient := discogs.GetTestClient()
	baseClient.UserId = 123
	baseClient.AddCollectionRelease(&pbd.Release{
		Id:         4243427,
		InstanceId: 1001,
		Title:      "Stronger Faster Science",
	})

	client := &statsTestDiscogsClient{
		TestDiscogsClient: baseClient,
		statsResponse: map[int64]*pbd.ReleaseStats{
			4243427: {
				LowPrice:    581,
				MedianPrice: 1855,
				HighPrice:   3082,
			},
		},
	}

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test-token"},
	}
	err := b.db.SaveUser(ctx, su)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	initialRecord := &pb.Record{
		Release: &pbd.Release{
			Id:         4243427,
			InstanceId: 1001,
			Title:      "Stronger Faster Science",
		},
	}
	err = b.db.SaveRecord(ctx, 123, initialRecord, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save initial record: %v", err)
	}

	err = b.RefreshRelease(ctx, 1001, su, client, true)
	if err != nil {
		t.Fatalf("RefreshRelease returned unexpected error: %v", err)
	}

	record, err := b.db.GetRecord(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("Failed to retrieve record from db: %v", err)
	}

	if record.GetLowPrice().GetValue() != 581 {
		t.Errorf("Expected LowPrice 581, got %v", record.GetLowPrice().GetValue())
	}
	if record.GetMedianPrice().GetValue() != 1855 {
		t.Errorf("Expected MedianPrice 1855, got %v", record.GetMedianPrice().GetValue())
	}
	if record.GetHighPrice().GetValue() != 3082 {
		t.Errorf("Expected HighPrice 3082, got %v", record.GetHighPrice().GetValue())
	}
	if record.GetLastStatRefresh() == 0 {
		t.Errorf("Expected LastStatRefresh to be set, got 0")
	}
}

func TestRefreshRelease_NotFoundFallback(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	baseClient := discogs.GetTestClient()
	baseClient.UserId = 123
	baseClient.AddCollectionRelease(&pbd.Release{
		Id:         99999,
		InstanceId: 2002,
		Title:      "Unsold Release",
	})

	client := &statsTestDiscogsClient{
		TestDiscogsClient: baseClient,
		statsError: map[int64]error{
			99999: status.Errorf(codes.NotFound, "no price stats available"),
		},
	}

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test-token"},
	}
	err := b.db.SaveUser(ctx, su)
	if err != nil {
		t.Fatalf("Failed to save user: %v", err)
	}

	initialRecord := &pb.Record{
		Release: &pbd.Release{
			Id:         99999,
			InstanceId: 2002,
		},
	}
	err = b.db.SaveRecord(ctx, 123, initialRecord, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("Failed to save initial record: %v", err)
	}

	err = b.RefreshRelease(ctx, 2002, su, client, true)
	if err != nil {
		t.Fatalf("RefreshRelease returned unexpected error on NotFound stats: %v", err)
	}

	record, err := b.db.GetRecord(ctx, 123, 2002)
	if err != nil {
		t.Fatalf("Failed to retrieve record from db: %v", err)
	}

	if record.GetLowPrice().GetValue() != 500 {
		t.Errorf("Expected fallback LowPrice 500, got %v", record.GetLowPrice().GetValue())
	}
	if record.GetMedianPrice().GetValue() != 10000 {
		t.Errorf("Expected fallback MedianPrice 10000, got %v", record.GetMedianPrice().GetValue())
	}
	if record.GetHighPrice().GetValue() != 10000 {
		t.Errorf("Expected fallback HighPrice 10000, got %v", record.GetHighPrice().GetValue())
	}
	if record.GetLastStatRefresh() == 0 {
		t.Errorf("Expected LastStatRefresh to be set on NotFound fallback, got 0")
	}
}

func TestRefreshRelease_ErrorPropagation(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	baseClient := discogs.GetTestClient()
	baseClient.UserId = 123
	baseClient.AddCollectionRelease(&pbd.Release{
		Id:         88888,
		InstanceId: 3003,
	})

	client := &statsTestDiscogsClient{
		TestDiscogsClient: baseClient,
		statsError: map[int64]error{
			88888: status.Errorf(codes.Internal, "internal discogs error"),
		},
	}

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test-token"},
	}
	_ = b.db.SaveUser(ctx, su)
	_ = b.db.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         88888,
			InstanceId: 3003,
		},
	}, &db.SaveOptions{})

	err := b.RefreshRelease(ctx, 3003, su, client, true)
	if err == nil {
		t.Fatalf("Expected error propagating internal error, got nil")
	}
	if status.Code(err) != codes.Internal {
		t.Errorf("Expected codes.Internal, got %v", status.Code(err))
	}
}

func TestRefreshRelease_HandlerExecuteAndValidate(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	baseClient := discogs.GetTestClient()
	baseClient.UserId = 123
	baseClient.AddCollectionRelease(&pbd.Release{
		Id:         4243427,
		InstanceId: 4004,
	})

	client := &statsTestDiscogsClient{
		TestDiscogsClient: baseClient,
		statsResponse: map[int64]*pbd.ReleaseStats{
			4243427: {
				LowPrice:    581,
				MedianPrice: 1855,
				HighPrice:   3082,
			},
		},
	}

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test-token"},
	}
	_ = b.db.SaveUser(ctx, su)
	_ = b.db.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         4243427,
			InstanceId: 4004,
		},
	}, &db.SaveOptions{})

	handler := &refreshReleaseHandler{b: b}

	// Validate missing intention -> InvalidArgument
	err := handler.Validate(ctx, b.db, &pb.QueueElement{
		Auth: "test-token",
		Entry: &pb.QueueElement_RefreshRelease{
			RefreshRelease: &pb.RefreshRelease{
				Iid: 4004,
			},
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument for missing intention, got %v", err)
	}

	// Validate with intention -> nil
	elem := &pb.QueueElement{
		Auth: "test-token",
		Entry: &pb.QueueElement_RefreshRelease{
			RefreshRelease: &pb.RefreshRelease{
				Iid:       4004,
				Intention: "Manual Update",
			},
		},
	}
	err = handler.Validate(ctx, b.db, elem)
	if err != nil {
		t.Fatalf("Expected validation to succeed, got %v", err)
	}

	// Execute handler
	err = handler.Execute(ctx, client, su, elem, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Handler execution failed: %v", err)
	}

	record, err := b.db.GetRecord(ctx, 123, 4004)
	if err != nil {
		t.Fatalf("Failed to retrieve record: %v", err)
	}
	if record.GetMedianPrice().GetValue() != 1855 {
		t.Errorf("Expected median price 1855, got %v", record.GetMedianPrice().GetValue())
	}
}
