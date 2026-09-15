package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
)

func TestGetOrganisation(t *testing.T) {
	module := GetOrganisation()
	if module == nil {
		t.Fatalf("GetOrganisation returned nil")
	}
	if module.Command != "org" {
		t.Errorf("Expected command 'org', got %v", module.Command)
	}
}

func TestResolvePlacement_CacheHit(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	// Pre-populate cache with instance 12345
	mgr.mu.Lock()
	mgr.data.InstanceCache[12345] = &pb.RecordCacheEntry{
		ArtistTitle:       "The Beatles - Abbey Road",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.mu.Unlock()

	mock := &mockGramophileClient{}
	placement := &pb.Placement{
		Iid:   12345,
		Width: 1.25,
	}

	got, err := resolvePlacement(ctx, mock, placement, mgr, false)
	if err != nil {
		t.Fatalf("resolvePlacement failed: %v", err)
	}

	expected := "The Beatles - Abbey Road"
	if got != expected {
		t.Errorf("resolvePlacement cache hit mismatch: got %q, want %q", got, expected)
	}

	if calls := mock.CallCount(); calls != 0 {
		t.Errorf("Expected 0 RPC calls on cache hit, got %d", calls)
	}
}

func TestResolvePlacement_CacheHit_WithDebug(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mgr.mu.Lock()
	mgr.data.InstanceCache[12345] = &pb.RecordCacheEntry{
		ArtistTitle:       "The Beatles - Abbey Road",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.mu.Unlock()

	mock := &mockGramophileClient{}
	placement := &pb.Placement{
		Iid:           12345,
		Width:         1.25,
		OriginalIndex: 4,
		Observations:  "test-obs",
		Space:         "Shelf-A",
	}

	got, err := resolvePlacement(ctx, mock, placement, mgr, true)
	if err != nil {
		t.Fatalf("resolvePlacement failed: %v", err)
	}

	expected := "The Beatles - Abbey Road {4 - test-obs (Shelf-A)}"
	if got != expected {
		t.Errorf("resolvePlacement with debug mismatch: got %q, want %q", got, expected)
	}

	if calls := mock.CallCount(); calls != 0 {
		t.Errorf("Expected 0 RPC calls on cache hit, got %d", calls)
	}
}

func TestResolvePlacement_CacheMiss(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id:         888,
								InstanceId: 67890,
								Title:      "The Dark Side of the Moon",
								Artists: []*pbd.Artist{
									{Name: "Pink Floyd"},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	placement := &pb.Placement{
		Iid:   67890,
		Width: 2.0,
	}

	got, err := resolvePlacement(ctx, mock, placement, mgr, false)
	if err != nil {
		t.Fatalf("resolvePlacement failed: %v", err)
	}

	expected := "Pink Floyd - The Dark Side of the Moon"
	if got != expected {
		t.Errorf("resolvePlacement cache miss mismatch: got %q, want %q", got, expected)
	}

	if calls := mock.CallCount(); calls != 1 {
		t.Errorf("Expected 1 RPC call on cache miss, got %d", calls)
	}

	// Verify it was persisted to cache and next lookup produces 0 additional calls
	got2, err := resolvePlacement(ctx, mock, placement, mgr, false)
	if err != nil {
		t.Fatalf("resolvePlacement 2nd call failed: %v", err)
	}
	if got2 != expected {
		t.Errorf("2nd resolvePlacement mismatch: got %q, want %q", got2, expected)
	}
	if calls := mock.CallCount(); calls != 1 {
		t.Errorf("Expected CallCount to remain 1 after cache hit, got %d", calls)
	}
}

func TestResolvePlacement_NilCacheFallback(t *testing.T) {
	ctx := context.Background()
	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Width: 1.5,
							Release: &pbd.Release{
								Id:         101,
								InstanceId: 999,
								Title:      "A Night at the Opera",
								DateAdded:  1700000000 * int64(time.Second),
								Artists: []*pbd.Artist{
									{Name: "Queen"},
								},
							},
						},
					},
				},
			}, nil
		},
	}

	placement := &pb.Placement{
		Iid:   999,
		Width: 1.5,
	}

	got, err := resolvePlacement(ctx, mock, placement, nil, false)
	if err != nil {
		t.Fatalf("resolvePlacement failed with nil cache: %v", err)
	}

	expectedPrefix := "Queen - A Night at the Opera [1.5 / 1.5] "
	if !strings.HasPrefix(got, expectedPrefix) {
		t.Errorf("Expected prefix %q, got %q", expectedPrefix, got)
	}
	if mock.CallCount() != 1 {
		t.Errorf("Expected 1 RPC call for nil cache, got %d", mock.CallCount())
	}
}

func TestRenderOrgPlacements_NonInteractive(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	// Pre-populate instance 10
	mgr.mu.Lock()
	mgr.data.InstanceCache[10] = &pb.RecordCacheEntry{
		ArtistTitle:       "Artist One - Album One",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.mu.Unlock()

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			iid := in.GetGetRecordWithId().GetInstanceId()
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								InstanceId: iid,
								Title:      "Album Two",
								Artists:    []*pbd.Artist{{Name: "Artist Two"}},
							},
						},
					},
				},
			}, nil
		},
	}

	placements := []*pb.Placement{
		{
			Iid:   10,
			Space: "Shelf1",
			Unit:  1,
			Width: 1.0,
		},
		{
			Iid:   20,
			Space: "Shelf1",
			Unit:  1,
			Width: 1.5,
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(false))

	totalWidth, err := renderOrgPlacements(ctx, mock, placements, -1, false, mgr, renderer)
	if err != nil {
		t.Fatalf("renderOrgPlacements failed: %v", err)
	}

	if totalWidth != 2.5 {
		t.Errorf("Expected total width 2.5, got %v", totalWidth)
	}

	output := buf.String()
	// Must not contain any ANSI escape characters
	if strings.Contains(output, "\033") || strings.Contains(output, "\r") {
		t.Errorf("Non-interactive output must not contain ANSI escape sequences: %q", output)
	}

	expectedLines := []string{
		"0. [Shelf1-1] Artist One - Album One",
		"1. [Shelf1-1] Artist Two - Album Two",
	}

	for _, expectedLine := range expectedLines {
		if !strings.Contains(output, expectedLine) {
			t.Errorf("Expected output to contain line %q, got output:\n%s", expectedLine, output)
		}
	}
}

func TestRenderOrgPlacements_InteractiveTTY(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			time.Sleep(20 * time.Millisecond)
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								InstanceId: 30,
								Title:      "TTY Album",
								Artists:    []*pbd.Artist{{Name: "TTY Artist"}},
							},
						},
					},
				},
			}, nil
		},
	}

	placements := []*pb.Placement{
		{
			Iid:   30,
			Space: "Shelf2",
			Unit:  2,
			Width: 1.0,
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(true), WithInterval(10*time.Millisecond))

	totalWidth, err := renderOrgPlacements(ctx, mock, placements, -1, false, mgr, renderer)
	if err != nil {
		t.Fatalf("renderOrgPlacements failed: %v", err)
	}

	if totalWidth != 1.0 {
		t.Errorf("Expected totalWidth 1.0, got %v", totalWidth)
	}

	output := buf.String()
	// TTY output should contain ANSI clear line sequences
	if !strings.Contains(output, AnsiClearLine) {
		t.Errorf("Expected ANSI clear sequence in TTY mode, got: %q", output)
	}
	if !strings.Contains(output, "0. [Shelf2-2] TTY Artist - TTY Album") {
		t.Errorf("Expected finalized text in output, got: %q", output)
	}
}

func TestRenderOrgPlacements_SlotFiltering(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mgr.mu.Lock()
	mgr.data.InstanceCache[1] = &pb.RecordCacheEntry{ArtistTitle: "A - One", ResolvedTimestamp: time.Now().Unix()}
	mgr.data.InstanceCache[2] = &pb.RecordCacheEntry{ArtistTitle: "B - Two", ResolvedTimestamp: time.Now().Unix()}
	mgr.mu.Unlock()

	mock := &mockGramophileClient{}

	placements := []*pb.Placement{
		{Iid: 1, Space: "S1", Unit: 1, Width: 1.0},
		{Iid: 2, Space: "S1", Unit: 2, Width: 2.0},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(false))

	// Request slot 2 only
	totalWidth, err := renderOrgPlacements(ctx, mock, placements, 2, false, mgr, renderer)
	if err != nil {
		t.Fatalf("renderOrgPlacements failed: %v", err)
	}

	if totalWidth != 2.0 {
		t.Errorf("Expected width 2.0 for slot 2, got %v", totalWidth)
	}

	output := buf.String()
	if strings.Contains(output, "A - One") {
		t.Errorf("Did not expect slot 1 record in output when filtering to slot 2")
	}
	if !strings.Contains(output, "B - Two") {
		t.Errorf("Expected slot 2 record in output, got:\n%s", output)
	}
}
