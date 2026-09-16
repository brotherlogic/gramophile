package main

import (
	"bytes"
	"context"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetLocate(t *testing.T) {
	module := GetLocate()
	if module == nil {
		t.Fatalf("GetLocate returned nil")
	}
	if module.Command != "locate" {
		t.Errorf("Expected command 'locate', got %v", module.Command)
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		before int
		after  int
		want   float64
	}{
		{0, 0, 100.0},
		{1, 1, 66.66666666666666},
		{0, 1, 50.0},
		{0, 1, 50.0},
		{1, 0, 100.0},
		{9, 0, 100.0},
		{0, 9, 10.0},
		{4, 5, 50.0},
	}

	for _, tt := range tests {
		got := calculatePercentage(tt.before, tt.after)
		if got != tt.want {
			t.Errorf("calculatePercentage(%v, %v) = %v, want %v", tt.before, tt.after, got, tt.want)
		}
	}
}

func TestFormatLocationOutput(t *testing.T) {
	loc := &pb.Location{
		LocationName: "Vinyl Rack",
		Slot:         12,
		Shelf:        "Shelf 1",
		Record:       "The Beatles - Abbey Road",
		Before: []*pb.Context{
			{Record: "Pink Floyd - Dark Side of the Moon"},
		},
		After: []*pb.Context{
			{Record: "Queen - A Night at the Opera"},
		},
	}

	got := formatLocationOutput(loc)
	expected := "The Beatles - Abbey Road is in Vinyl Rack, Slot 12 (67 %):\n\nPink Floyd - Dark Side of the Moon\nThe Beatles - Abbey Road\nQueen - A Night at the Opera\n\n"

	if got != expected {
		t.Errorf("formatLocationOutput mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}

func TestFormatLocationOutput_Nil(t *testing.T) {
	if got := formatLocationOutput(nil); got != "" {
		t.Errorf("Expected empty string for nil location, got %q", got)
	}
}

type mockLocateClient struct {
	pb.GramophileEServiceClient
	mu             sync.Mutex
	getRecordCalls int
	locateCalls    int
	getRecordFunc  func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error)
	locateFunc     func(ctx context.Context, in *pb.LocateRecordRequest, opts ...grpc.CallOption) (*pb.LocateRecordResponse, error)
}

func (m *mockLocateClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	m.mu.Lock()
	m.getRecordCalls++
	m.mu.Unlock()
	if m.getRecordFunc != nil {
		return m.getRecordFunc(ctx, in, opts...)
	}
	return nil, status.Errorf(codes.Unimplemented, "unimplemented")
}

func (m *mockLocateClient) LocateRecord(ctx context.Context, in *pb.LocateRecordRequest, opts ...grpc.CallOption) (*pb.LocateRecordResponse, error) {
	m.mu.Lock()
	m.locateCalls++
	m.mu.Unlock()
	if m.locateFunc != nil {
		return m.locateFunc(ctx, in, opts...)
	}
	return nil, status.Errorf(codes.Unimplemented, "unimplemented")
}

func (m *mockLocateClient) GetRecordCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.getRecordCalls
}

func (m *mockLocateClient) LocateCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.locateCalls
}

func TestFormatLocationOutput_WithCache_Populate(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	loc := &pb.Location{
		LocationName: "Vinyl Rack",
		Slot:         12,
		Shelf:        "Shelf 1",
		Record:       "The Beatles - Abbey Road",
		Before: []*pb.Context{
			{Iid: 101, Record: "Pink Floyd - Dark Side of the Moon"},
		},
		After: []*pb.Context{
			{Iid: 102, Record: "Queen - A Night at the Opera"},
		},
	}

	got := formatLocationOutput(loc, mgr, int64(999))
	expected := "The Beatles - Abbey Road is in Vinyl Rack, Slot 12 (67 %):\n\nPink Floyd - Dark Side of the Moon\nThe Beatles - Abbey Road\nQueen - A Night at the Opera\n\n"

	if got != expected {
		t.Errorf("formatLocationOutput mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}

	// Verify cache was populated
	if !mgr.HasInstance(101) {
		t.Errorf("Expected instance 101 to be populated in cache")
	}
	if !mgr.HasInstance(102) {
		t.Errorf("Expected instance 102 to be populated in cache")
	}
	if !mgr.HasRelease(999) {
		t.Errorf("Expected release 999 to be populated in cache")
	}
}

func TestFormatLocationOutput_WithCache_QueryHit(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	// Pre-populate with cached metadata differing from fallback location strings
	mgr.mu.Lock()
	mgr.data.ReleaseCache[999] = &pb.RecordCacheEntry{
		ArtistTitle:       "The Beatles - Abbey Road (2019 Mix)",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.data.InstanceCache[101] = &pb.RecordCacheEntry{
		ArtistTitle:       "Pink Floyd - Dark Side of the Moon (50th Ann)",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.data.InstanceCache[102] = &pb.RecordCacheEntry{
		ArtistTitle:       "Queen - A Night at the Opera (Remaster)",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.mu.Unlock()

	loc := &pb.Location{
		LocationName: "Vinyl Rack",
		Slot:         12,
		Shelf:        "Shelf 1",
		Record:       "Fallback Beatles",
		Before: []*pb.Context{
			{Iid: 101, Record: "Fallback Pink Floyd"},
		},
		After: []*pb.Context{
			{Iid: 102, Record: "Fallback Queen"},
		},
	}

	got := formatLocationOutput(loc, mgr, int64(999))
	expected := "The Beatles - Abbey Road (2019 Mix) is in Vinyl Rack, Slot 12 (67 %):\n\nPink Floyd - Dark Side of the Moon (50th Ann)\nThe Beatles - Abbey Road (2019 Mix)\nQueen - A Night at the Opera (Remaster)\n\n"

	if got != expected {
		t.Errorf("formatLocationOutput cache query hit mismatch.\nGot:\n%q\nWant:\n%q", got, expected)
	}
}

func TestRenderLocation_InteractiveTTY(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mock := &mockLocateClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			time.Sleep(10 * time.Millisecond)
			iid := in.GetGetRecordWithId().GetInstanceId()
			relId := in.GetGetRecordWithId().GetReleaseId()
			title := "Unknown"
			artist := "Various"
			if relId == 500 {
				title = "Target Album"
				artist = "Target Artist"
			} else if iid == 201 {
				title = "Before Album"
				artist = "Before Artist"
			} else if iid == 202 {
				title = "After Album"
				artist = "After Artist"
			}
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id:         relId,
								InstanceId: iid,
								Title:      title,
								Artists:    []*pbd.Artist{{Name: artist}},
							},
						},
					},
				},
			}, nil
		},
	}

	loc := &pb.Location{
		LocationName: "Main Shelf",
		Slot:         5,
		Before: []*pb.Context{
			{Iid: 201},
		},
		After: []*pb.Context{
			{Iid: 202},
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(true), WithInterval(5*time.Millisecond))

	out, err := renderLocation(ctx, mock, loc, 500, mgr, renderer)
	if err != nil {
		t.Fatalf("renderLocation failed: %v", err)
	}

	rawOut := buf.String()
	// TTY output should contain ANSI clear line sequences
	if !strings.Contains(rawOut, AnsiClearLine) {
		t.Errorf("Expected ANSI clear line sequence in TTY mode, got: %q", rawOut)
	}

	expectedSnippet := "Target Artist - Target Album is in Main Shelf, Slot 5 (67 %):"
	if !strings.Contains(out, expectedSnippet) {
		t.Errorf("Expected output to contain %q, got: %q", expectedSnippet, out)
	}
	if !strings.Contains(out, "Before Artist - Before Album") {
		t.Errorf("Expected output to contain before record, got: %q", out)
	}
	if !strings.Contains(out, "After Artist - After Album") {
		t.Errorf("Expected output to contain after record, got: %q", out)
	}
}

func TestRenderLocation_NonInteractive(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mock := &mockLocateClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			iid := in.GetGetRecordWithId().GetInstanceId()
			relId := in.GetGetRecordWithId().GetReleaseId()
			title := "Clean Target"
			artist := "Clean Artist"
			if iid == 301 {
				title = "Clean Context"
				artist = "Context Artist"
			}
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id:         relId,
								InstanceId: iid,
								Title:      title,
								Artists:    []*pbd.Artist{{Name: artist}},
							},
						},
					},
				},
			}, nil
		},
	}

	loc := &pb.Location{
		LocationName: "Clean Shelf",
		Slot:         1,
		Before: []*pb.Context{
			{Iid: 301},
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(false))

	out, err := renderLocation(ctx, mock, loc, 600, mgr, renderer)
	if err != nil {
		t.Fatalf("renderLocation failed: %v", err)
	}

	rawOut := buf.String()
	if strings.Contains(rawOut, "\033") || strings.Contains(rawOut, "\r") {
		t.Errorf("Non-interactive output must not contain ANSI escape sequences: %q", rawOut)
	}

	expectedHeader := "Clean Artist - Clean Target is in Clean Shelf, Slot 1 (100 %):\n\nContext Artist - Clean Context\nClean Artist - Clean Target\n\n"
	if out != expectedHeader {
		t.Errorf("renderLocation non-interactive mismatch.\nGot:\n%q\nWant:\n%q", out, expectedHeader)
	}
}

func TestRenderLocation_CacheHit_NoRPC(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mgr.mu.Lock()
	mgr.data.ReleaseCache[700] = &pb.RecordCacheEntry{
		ArtistTitle:       "Cached Artist - Cached Target",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.data.InstanceCache[401] = &pb.RecordCacheEntry{
		ArtistTitle:       "Cached Artist - Cached Before",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.mu.Unlock()

	mock := &mockLocateClient{}

	loc := &pb.Location{
		LocationName: "Cached Shelf",
		Slot:         2,
		Before: []*pb.Context{
			{Iid: 401},
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(false))

	out, err := renderLocation(ctx, mock, loc, 700, mgr, renderer)
	if err != nil {
		t.Fatalf("renderLocation failed: %v", err)
	}

	if mock.GetRecordCallCount() != 0 {
		t.Errorf("Expected 0 RPC calls on complete cache hit, got %d", mock.GetRecordCallCount())
	}

	expected := "Cached Artist - Cached Target is in Cached Shelf, Slot 2 (100 %):\n\nCached Artist - Cached Before\nCached Artist - Cached Target\n\n"
	if out != expected {
		t.Errorf("renderLocation output mismatch.\nGot:\n%q\nWant:\n%q", out, expected)
	}
}

func TestRunLocate_EndToEnd(t *testing.T) {
	ctx := context.Background()
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mock := &mockLocateClient{
		locateFunc: func(ctx context.Context, in *pb.LocateRecordRequest, opts ...grpc.CallOption) (*pb.LocateRecordResponse, error) {
			if in.GetReleaseId() != 12345 {
				return nil, status.Errorf(codes.NotFound, "release not found")
			}
			return &pb.LocateRecordResponse{
				Locations: []*pb.Location{
					{
						LocationName: "E2E Shelf",
						Slot:         3,
						Record:       "E2E Artist - E2E Target",
						Before: []*pb.Context{
							{Iid: 901, Record: "E2E Artist - Context 901"},
						},
					},
				},
			}, nil
		},
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			return &pb.GetRecordResponse{
				Records: []*pb.RecordResponse{
					{
						Record: &pb.Record{
							Release: &pbd.Release{
								Id:         12345,
								Title:      "E2E Target",
								Artists:    []*pbd.Artist{{Name: "E2E Artist"}},
							},
						},
					},
				},
			}, nil
		},
	}

	var buf bytes.Buffer
	renderer := NewTerminalRenderer(&buf, WithTTY(false))

	err = runLocate(ctx, mock, []string{"-id", "12345"}, mgr, renderer)
	if err != nil {
		t.Fatalf("runLocate failed: %v", err)
	}

	if mock.LocateCallCount() != 1 {
		t.Errorf("Expected 1 LocateRecord call, got %d", mock.LocateCallCount())
	}

	output := buf.String()
	if !strings.Contains(output, "E2E Artist - E2E Target is in E2E Shelf, Slot 3 (100 %):") {
		t.Errorf("Output missing expected header, got:\n%s", output)
	}

	// Verify second run utilizes cache for instances
	callsBeforeSecondRun := mock.GetRecordCallCount()
	var buf2 bytes.Buffer
	renderer2 := NewTerminalRenderer(&buf2, WithTTY(false))
	err = runLocate(ctx, mock, []string{"-id", "12345"}, mgr, renderer2)
	if err != nil {
		t.Fatalf("second runLocate failed: %v", err)
	}
	if mock.GetRecordCallCount() != callsBeforeSecondRun {
		t.Errorf("Expected no additional GetRecord calls on second run with cache, got %d calls", mock.GetRecordCallCount()-callsBeforeSecondRun)
	}
}
