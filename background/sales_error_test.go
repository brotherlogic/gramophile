package background

import (
	"context"
	"errors"
	"fmt"
	"testing"

	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	"github.com/brotherlogic/gramophile/db"
	pstore_client "github.com/brotherlogic/pstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type failingGHClient struct {
	ghbclient.GithubridgeClient
}

func (f *failingGHClient) GetIssues(ctx context.Context, req *ghbpb.GetIssuesRequest) (*ghbpb.GetIssuesResponse, error) {
	return nil, errors.New("simulated githubridge connection failure")
}

func (f *failingGHClient) CreateIssue(ctx context.Context, req *ghbpb.CreateIssueRequest) (*ghbpb.CreateIssueResponse, error) {
	return nil, errors.New("simulated githubridge create failure")
}

func TestReportSaleAdjustmentError_CreatesIssue(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	err := b.reportSaleAdjustmentError(ctx, 12345, "adjustPrice", errors.New("database timeout"))
	if err != nil {
		t.Fatalf("expected nil error returned, got %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues from test client: %v", err)
	}

	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected 1 issue to be created, got %d", len(resp.GetIssues()))
	}

	issue := resp.GetIssues()[0]
	expectedTitle := "Sale Adjustment Failure: adjustPrice on sale 12345"
	if issue.GetTitle() != expectedTitle {
		t.Errorf("expected issue title %q, got %q", expectedTitle, issue.GetTitle())
	}
	if issue.GetRepo() != "gramophile" {
		t.Errorf("expected issue repo 'gramophile', got %q", issue.GetRepo())
	}
}

func TestReportSaleAdjustmentError_Deduplication(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// First call - should create issue
	err := b.reportSaleAdjustmentError(ctx, 12345, "adjustPrice", errors.New("database timeout"))
	if err != nil {
		t.Fatalf("first call failed: %v", err)
	}

	// Second call with same sid and action - should deduplicate
	err = b.reportSaleAdjustmentError(ctx, 12345, "adjustPrice", errors.New("database timeout again"))
	if err != nil {
		t.Fatalf("second call failed: %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}

	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected 1 issue after deduplication, got %d", len(resp.GetIssues()))
	}
}

func TestReportSaleAdjustmentError_DistinctIssuesForDifferentSalesOrActions(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Sale 100, action adjustPrice
	err := b.reportSaleAdjustmentError(ctx, 100, "adjustPrice", errors.New("adjust err"))
	if err != nil {
		t.Fatalf("call 1 failed: %v", err)
	}

	// Sale 200, action adjustPrice
	err = b.reportSaleAdjustmentError(ctx, 200, "adjustPrice", errors.New("adjust err 2"))
	if err != nil {
		t.Fatalf("call 2 failed: %v", err)
	}

	// Sale 100, action SaveSale
	err = b.reportSaleAdjustmentError(ctx, 100, "SaveSale", errors.New("save err"))
	if err != nil {
		t.Fatalf("call 3 failed: %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}

	if len(resp.GetIssues()) != 3 {
		t.Fatalf("expected 3 distinct issues, got %d", len(resp.GetIssues()))
	}
}

func TestReportSaleAdjustmentError_HandlesClientErrorGracefully(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	b.ghclient = &failingGHClient{}

	// Should log the error and not fail the process
	err := b.reportSaleAdjustmentError(ctx, 12345, "adjustPrice", errors.New("some error"))
	if err != nil {
		t.Errorf("expected reportSaleAdjustmentError to handle client failure gracefully and return nil, got: %v", err)
	}
}

func TestIsUnavailable(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "standard Go error",
			err:      errors.New("standard error"),
			expected: false,
		},
		{
			name:     "direct gRPC unavailable",
			err:      status.Error(codes.Unavailable, "service unavailable"),
			expected: true,
		},
		{
			name:     "wrapped gRPC unavailable with fmt.Errorf",
			err:      fmt.Errorf("failed calling rpc: %w", status.Error(codes.Unavailable, "temporary outage")),
			expected: true,
		},
		{
			name:     "deeply wrapped gRPC unavailable",
			err:      fmt.Errorf("outer wrap: %w", fmt.Errorf("inner wrap: %w", status.Error(codes.Unavailable, "temporary outage"))),
			expected: true,
		},
		{
			name:     "joined error containing gRPC unavailable",
			err:      errors.Join(errors.New("some other error"), status.Error(codes.Unavailable, "unavailable")),
			expected: true,
		},
		{
			name:     "direct gRPC internal error",
			err:      status.Error(codes.Internal, "internal server error"),
			expected: false,
		},
		{
			name:     "direct gRPC invalid argument error",
			err:      status.Error(codes.InvalidArgument, "invalid input"),
			expected: false,
		},
		{
			name:     "wrapped gRPC not found error",
			err:      fmt.Errorf("wrapped: %w", status.Error(codes.NotFound, "record not found")),
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isUnavailable(tc.err)
			if got != tc.expected {
				t.Errorf("isUnavailable(%v) = %v, expected %v", tc.err, got, tc.expected)
			}
		})
	}
}

func TestReportSaleAdjustmentError_SuppressesUnavailableOnGetSale(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Direct Unavailable error on GetSale
	err := b.reportSaleAdjustmentError(ctx, 12345, "GetSale", status.Error(codes.Unavailable, "service unavailable"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Wrapped Unavailable error on GetSale
	wrappedErr := fmt.Errorf("rpc failure: %w", status.Error(codes.Unavailable, "backend down"))
	err = b.reportSaleAdjustmentError(ctx, 12346, "GetSale", wrappedErr)
	if err != nil {
		t.Fatalf("expected nil error on wrapped err, got %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}

	if len(resp.GetIssues()) != 0 {
		t.Fatalf("expected 0 issues created for Unavailable on GetSale, got %d", len(resp.GetIssues()))
	}
}

func TestReportSaleAdjustmentError_ReportsNonUnavailableOnGetSale(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Internal error on GetSale
	err := b.reportSaleAdjustmentError(ctx, 12345, "GetSale", status.Error(codes.Internal, "internal server error"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// InvalidArgument error on GetSale
	err = b.reportSaleAdjustmentError(ctx, 12346, "GetSale", status.Error(codes.InvalidArgument, "invalid argument"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}

	if len(resp.GetIssues()) != 2 {
		t.Fatalf("expected 2 issues created for non-Unavailable on GetSale, got %d", len(resp.GetIssues()))
	}
}

func TestReportSaleAdjustmentError_ReportsUnavailableOnOtherActions(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Unavailable error on adjustPrice
	err := b.reportSaleAdjustmentError(ctx, 12345, "adjustPrice", status.Error(codes.Unavailable, "service unavailable"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	// Unavailable error on SaveSale
	err = b.reportSaleAdjustmentError(ctx, 12346, "SaveSale", status.Error(codes.Unavailable, "service unavailable"))
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues: %v", err)
	}

	if len(resp.GetIssues()) != 2 {
		t.Fatalf("expected 2 issues created for Unavailable on other actions, got %d", len(resp.GetIssues()))
	}
}
