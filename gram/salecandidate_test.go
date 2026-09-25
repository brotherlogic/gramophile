package main

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type mockSaleCandidateClient struct {
	pb.GramophileEServiceClient
	lastReq *pb.GetRecordRequest
	resp    *pb.GetRecordResponse
	err     error
}

func (m *mockSaleCandidateClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	m.lastReq = in
	if m.err != nil {
		return nil, m.err
	}
	return m.resp, nil
}

func TestGetSaleCandidate(t *testing.T) {
	module := GetSaleCandidate()
	if module == nil {
		t.Fatalf("GetSaleCandidate returned nil")
	}
	if module.Command != "salecandidate" {
		t.Errorf("Expected command 'salecandidate', got %q", module.Command)
	}
	if module.Execute == nil {
		t.Errorf("Expected Execute function to be set")
	}
}

func TestFormatSaleCandidate_CompleteMetadata(t *testing.T) {
	arrivedTime := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	record := &pb.Record{
		Release: &pbd.Release{
			Id:         1001,
			InstanceId: 2001,
			Title:      "Kind of Blue",
			Rating:     5,
			Artists: []*pbd.Artist{
				{Name: "Miles Davis"},
				{Name: "John Coltrane"},
			},
			DateAdded: arrivedTime.Add(-24 * time.Hour).UnixNano(),
		},
		PackageScore: 4,
		MedianPrice: &pbd.Price{
			Currency: "USD",
			Value:    3550, // $35.50
		},
		Arrived: arrivedTime.UnixNano(),
	}

	output := formatSaleCandidate(record)

	expectedSubstrings := []string{
		"Miles Davis, John Coltrane",
		"Kind of Blue",
		"5",
		"4",
		"35.50",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(output, sub) {
			t.Errorf("Expected formatted output to contain %q, but got:\n%s", sub, output)
		}
	}

	if !strings.Contains(output, "Artist:") ||
		!strings.Contains(output, "Title:") ||
		!strings.Contains(output, "Discogs Rating:") ||
		!strings.Contains(output, "Package Score:") ||
		!strings.Contains(output, "Median Price:") ||
		!strings.Contains(output, "Arrival Date:") {
		t.Errorf("Expected output to contain all candidate metadata labels, got:\n%s", output)
	}
}

func TestFormatSaleCandidate_FallbackAndDefaults(t *testing.T) {
	addedTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	record := &pb.Record{
		Release: &pbd.Release{
			Id:         1002,
			InstanceId: 2002,
			Title:      "Untitled Album",
			Rating:     0,
			DateAdded:  addedTime.UnixNano(),
		},
		PackageScore: 0,
		// No Arrived set -> should fall back to DateAdded
		// No MedianPrice set -> should default to $0.00
	}

	output := formatSaleCandidate(record)

	if !strings.Contains(output, "NO_ARTIST") {
		t.Errorf("Expected NO_ARTIST when no artists present, got:\n%s", output)
	}
	if !strings.Contains(output, "Untitled Album") {
		t.Errorf("Expected title in output, got:\n%s", output)
	}
	if !strings.Contains(output, "0.00") {
		t.Errorf("Expected $0.00 for missing median price, got:\n%s", output)
	}
}

func TestRunSaleCandidate_Success(t *testing.T) {
	mock := &mockSaleCandidateClient{
		resp: &pb.GetRecordResponse{
			Records: []*pb.RecordResponse{
				{
					Record: &pb.Record{
						Release: &pbd.Release{
							Title:  "Blue Train",
							Rating: 4,
							Artists: []*pbd.Artist{
								{Name: "John Coltrane"},
							},
						},
						PackageScore: 3,
						MedianPrice: &pbd.Price{
							Currency: "USD",
							Value:    2500,
						},
						Arrived: time.Now().UnixNano(),
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := runSaleCandidate(context.Background(), mock, "Jazz", &buf)
	if err != nil {
		t.Fatalf("runSaleCandidate failed: %v", err)
	}

	if mock.lastReq == nil {
		t.Fatalf("GetRecord was not called")
	}
	saleCandReq := mock.lastReq.GetGetSaleCandidate()
	if saleCandReq == nil {
		t.Fatalf("Expected GetSaleCandidate request, got: %v", mock.lastReq)
	}
	if saleCandReq.GetOrgName() != "Jazz" {
		t.Errorf("Expected OrgName 'Jazz', got %q", saleCandReq.GetOrgName())
	}

	output := buf.String()
	if !strings.Contains(output, "Blue Train") || !strings.Contains(output, "John Coltrane") {
		t.Errorf("Expected buffer to contain candidate output, got:\n%s", output)
	}
}

func TestRunSaleCandidate_Errors(t *testing.T) {
	// 1. Missing org name
	mock := &mockSaleCandidateClient{}
	var buf bytes.Buffer
	err := runSaleCandidate(context.Background(), mock, "", &buf)
	if err == nil {
		t.Errorf("Expected error for empty org name, got nil")
	}

	// 2. RPC Error
	mockErr := &mockSaleCandidateClient{
		err: status.Errorf(codes.NotFound, "organisation not found"),
	}
	err = runSaleCandidate(context.Background(), mockErr, "MissingOrg", &buf)
	if err == nil {
		t.Errorf("Expected error for RPC failure, got nil")
	}

	// 3. Empty Records response
	mockEmpty := &mockSaleCandidateClient{
		resp: &pb.GetRecordResponse{
			Records: []*pb.RecordResponse{},
		},
	}
	err = runSaleCandidate(context.Background(), mockEmpty, "EmptyOrg", &buf)
	if err == nil {
		t.Errorf("Expected error for empty records response, got nil")
	}
}
