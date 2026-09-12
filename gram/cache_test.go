package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type mockGramophileClient struct {
	pb.GramophileEServiceClient
	mu            sync.Mutex
	calls         int
	getRecordFunc func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error)
}

func (m *mockGramophileClient) GetRecord(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
	m.mu.Lock()
	m.calls++
	m.mu.Unlock()
	if m.getRecordFunc != nil {
		return m.getRecordFunc(ctx, in, opts...)
	}
	return nil, status.Errorf(codes.Unimplemented, "unimplemented")
}

func (m *mockGramophileClient) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.calls
}

func createTestRecordResponse(artist, title string, iid, releaseId int64) *pb.GetRecordResponse {
	return &pb.GetRecordResponse{
		Records: []*pb.RecordResponse{
			{
				Record: &pb.Record{
					Release: &pbd.Release{
						Id:         releaseId,
						InstanceId: iid,
						Title:      title,
						Artists: []*pbd.Artist{
							{Name: artist},
						},
					},
				},
			},
		},
	}
}

func TestCache_ColdStartAndCreation(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "test_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	if _, err := os.Stat(cachePath); !os.IsNotExist(err) {
		t.Fatalf("Expected cache file to not exist yet, but stat err was %v", err)
	}

	if err := mgr.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("Stat failed after save: %v", err)
	}

	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("Expected permissions 0600, got %#o", perm)
	}

	mgr2, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("Reloading cache failed: %v", err)
	}
	if len(mgr2.data.GetInstanceCache()) != 0 || len(mgr2.data.GetReleaseCache()) != 0 {
		t.Errorf("Expected empty cache after reload, got instances=%d, releases=%d",
			len(mgr2.data.GetInstanceCache()), len(mgr2.data.GetReleaseCache()))
	}
}

func TestCache_FreshHit(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "fresh_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}

	mgr.data.InstanceCache[101] = &pb.RecordCacheEntry{
		ArtistTitle:       "Pink Floyd - The Wall",
		ResolvedTimestamp: time.Now().Unix(),
	}
	mgr.data.ReleaseCache[201] = &pb.RecordCacheEntry{
		ArtistTitle:       "Pink Floyd - Animals",
		ResolvedTimestamp: time.Now().Unix(),
	}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			t.Fatalf("GetRecord should not be called on a fresh hit")
			return nil, nil
		},
	}

	ctx := context.Background()
	val, err := mgr.ResolveInstance(ctx, mock, 101)
	if err != nil {
		t.Fatalf("ResolveInstance failed: %v", err)
	}
	if val != "Pink Floyd - The Wall" {
		t.Errorf("Expected 'Pink Floyd - The Wall', got %q", val)
	}

	valRel, err := mgr.ResolveRelease(ctx, mock, 201)
	if err != nil {
		t.Fatalf("ResolveRelease failed: %v", err)
	}
	if valRel != "Pink Floyd - Animals" {
		t.Errorf("Expected 'Pink Floyd - Animals', got %q", valRel)
	}

	if mock.CallCount() != 0 {
		t.Errorf("Expected 0 remote calls on fresh hit, got %d", mock.CallCount())
	}
}

func TestCache_StaleWhileRevalidate(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "stale_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}
	mgr.retryBackoffs = []time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}

	// 8 days old
	staleTimestamp := time.Now().Add(-8 * 24 * time.Hour).Unix()
	mgr.data.InstanceCache[102] = &pb.RecordCacheEntry{
		ArtistTitle:       "Old Artist - Old Title",
		ResolvedTimestamp: staleTimestamp,
	}

	revalidateDone := make(chan struct{})
	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			defer close(revalidateDone)
			return createTestRecordResponse("New Artist", "New Title", 102, 202), nil
		},
	}

	ctx := context.Background()
	val, err := mgr.ResolveInstance(ctx, mock, 102)
	if err != nil {
		t.Fatalf("ResolveInstance failed: %v", err)
	}
	if val != "Old Artist - Old Title" {
		t.Fatalf("Expected immediate stale return 'Old Artist - Old Title', got %q", val)
	}

	select {
	case <-revalidateDone:
	case <-time.After(3 * time.Second):
		t.Fatalf("Timed out waiting for asynchronous revalidation")
	}

	// Small wait for Save() / cache update to settle
	time.Sleep(50 * time.Millisecond)

	mgr.mu.RLock()
	entry := mgr.data.GetInstanceCache()[102]
	mgr.mu.RUnlock()

	if entry == nil {
		t.Fatalf("Expected cache entry to exist")
	}
	if entry.GetArtistTitle() != "New Artist - New Title" {
		t.Errorf("Expected updated title 'New Artist - New Title', got %q", entry.GetArtistTitle())
	}
	if entry.GetResolvedTimestamp() <= staleTimestamp {
		t.Errorf("Expected fresh timestamp greater than staleTimestamp")
	}
}

func TestCache_MissAndSave(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "miss_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}
	mgr.retryBackoffs = []time.Duration{time.Millisecond, 2 * time.Millisecond, 3 * time.Millisecond}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			iid := in.GetGetRecordWithId().GetInstanceId()
			if iid == 103 {
				return createTestRecordResponse("Radiohead", "OK Computer", 103, 303), nil
			}
			relId := in.GetGetRecordWithId().GetReleaseId()
			if relId == 303 {
				return createTestRecordResponse("Radiohead", "Kid A", 104, 303), nil
			}
			return nil, status.Errorf(codes.NotFound, "not found")
		},
	}

	ctx := context.Background()
	val, err := mgr.ResolveInstance(ctx, mock, 103)
	if err != nil {
		t.Fatalf("ResolveInstance failed: %v", err)
	}
	if val != "Radiohead - OK Computer" {
		t.Errorf("Expected 'Radiohead - OK Computer', got %q", val)
	}

	valRel, err := mgr.ResolveRelease(ctx, mock, 303)
	if err != nil {
		t.Fatalf("ResolveRelease failed: %v", err)
	}
	if valRel != "Radiohead - Kid A" {
		t.Errorf("Expected 'Radiohead - Kid A', got %q", valRel)
	}

	// Reload from disk to verify persistence
	mgrReloaded, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("Failed to reload cache from disk: %v", err)
	}
	if entry := mgrReloaded.data.GetInstanceCache()[103]; entry == nil || entry.GetArtistTitle() != "Radiohead - OK Computer" {
		t.Errorf("Persisted instance entry mismatch: %v", entry)
	}
	if entry := mgrReloaded.data.GetReleaseCache()[303]; entry == nil || entry.GetArtistTitle() != "Radiohead - Kid A" {
		t.Errorf("Persisted release entry mismatch: %v", entry)
	}
}

func TestCache_CorruptFileRecovery(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "corrupt_cache")

	if err := os.WriteFile(cachePath, []byte("garbage binary data that cannot unmarshal"), 0600); err != nil {
		t.Fatalf("Failed to write corrupt cache file: %v", err)
	}

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager should not fail on corrupt file, got: %v", err)
	}

	if len(mgr.data.GetInstanceCache()) != 0 || len(mgr.data.GetReleaseCache()) != 0 {
		t.Errorf("Expected reset empty cache on corrupt file")
	}

	// Verify Save recovers and writes valid proto
	mgr.data.InstanceCache[999] = &pb.RecordCacheEntry{
		ArtistTitle:       "Recovered - Album",
		ResolvedTimestamp: time.Now().Unix(),
	}
	if err := mgr.Save(); err != nil {
		t.Fatalf("Save after recovery failed: %v", err)
	}

	raw, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("ReadFile failed: %v", err)
	}
	unmarshaled := &pb.RecordCache{}
	if err := proto.Unmarshal(raw, unmarshaled); err != nil {
		t.Fatalf("Saved file could not be unmarshaled: %v", err)
	}
	if unmarshaled.GetInstanceCache()[999].GetArtistTitle() != "Recovered - Album" {
		t.Errorf("Unmarshaled content mismatch: %v", unmarshaled)
	}
}

func TestCache_RetryAndBackoff(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "retry_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}
	mgr.retryBackoffs = []time.Duration{10 * time.Millisecond, 20 * time.Millisecond, 30 * time.Millisecond}

	var attempts int32
	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			att := atomic.AddInt32(&attempts, 1)
			if att < 3 {
				return nil, status.Errorf(codes.Unavailable, "transient error attempt %d", att)
			}
			return createTestRecordResponse("Retry Artist", "Retry Title", 104, 204), nil
		},
	}

	start := time.Now()
	ctx := context.Background()
	val, err := mgr.ResolveInstance(ctx, mock, 104)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("ResolveInstance failed: %v", err)
	}
	if val != "Retry Artist - Retry Title" {
		t.Errorf("Expected 'Retry Artist - Retry Title', got %q", val)
	}
	if atomic.LoadInt32(&attempts) != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
	if elapsed < 30*time.Millisecond {
		t.Errorf("Expected elapsed time >= 30ms (backoff sum), got %v", elapsed)
	}
}

func TestCache_UnresolvedExhaustion(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "unresolved_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}
	mgr.retryBackoffs = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			return nil, status.Errorf(codes.NotFound, "not found")
		},
	}

	ctx := context.Background()
	valIid, err := mgr.ResolveInstance(ctx, mock, 105)
	if err != nil {
		t.Fatalf("Expected nil error on unresolved fallback, got %v", err)
	}
	if valIid != "[Unresolved: 105]" {
		t.Errorf("Expected '[Unresolved: 105]', got %q", valIid)
	}

	valRel, err := mgr.ResolveRelease(ctx, mock, 205)
	if err != nil {
		t.Fatalf("Expected nil error on unresolved fallback, got %v", err)
	}
	if valRel != "[Unresolved: 205]" {
		t.Errorf("Expected '[Unresolved: 205]', got %q", valRel)
	}

	if count := mock.CallCount(); count != 6 { // 3 for instance, 3 for release
		t.Errorf("Expected 6 total calls (3 retries each), got %d", count)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "concurrent_cache")

	mgr, err := NewRecordCacheManager(cachePath)
	if err != nil {
		t.Fatalf("NewRecordCacheManager failed: %v", err)
	}
	mgr.retryBackoffs = []time.Duration{time.Millisecond, time.Millisecond, time.Millisecond}

	mock := &mockGramophileClient{
		getRecordFunc: func(ctx context.Context, in *pb.GetRecordRequest, opts ...grpc.CallOption) (*pb.GetRecordResponse, error) {
			time.Sleep(5 * time.Millisecond)
			iid := in.GetGetRecordWithId().GetInstanceId()
			relId := in.GetGetRecordWithId().GetReleaseId()
			if iid > 0 {
				return createTestRecordResponse(fmt.Sprintf("Artist %d", iid), fmt.Sprintf("Title %d", iid), iid, iid+1000), nil
			}
			return createTestRecordResponse(fmt.Sprintf("Artist Rel %d", relId), fmt.Sprintf("Title Rel %d", relId), relId+1000, relId), nil
		},
	}

	var wg sync.WaitGroup
	ctx := context.Background()
	for i := 0; i < 40; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			iid := int64(id % 10)
			relId := int64(id % 10)
			_, _ = mgr.ResolveInstance(ctx, mock, iid)
			_, _ = mgr.ResolveRelease(ctx, mock, relId)
			_ = mgr.Save()
		}(i)
	}

	wg.Wait()
}
