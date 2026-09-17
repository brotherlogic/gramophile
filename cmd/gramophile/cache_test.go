package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	protov2 "google.golang.org/protobuf/proto"
)

func createSampleRecord(iid int64, releaseID int64, artist string, title string) *pb.Record {
	return &pb.Record{
		Release: &pbd.Release{
			InstanceId: iid,
			Id:         releaseID,
			Title:      title,
			Artists: []*pbd.Artist{
				{Name: artist},
			},
		},
		LastUpdateTime: time.Now().Unix(),
	}
}

func TestCacheLoad_ValidDiskFile(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "cache.pb")

	now := time.Now().Unix()
	cache := &pb.CollectionCache{
		DiscogsUserId: 12345,
		LastSyncTime:  now,
		Records: []*pb.CachedRecord{
			{
				InstanceId:     101,
				ReleaseId:      201,
				Artist:         "Pink Floyd",
				Title:          "The Dark Side of the Moon",
				LastUpdatedTime: now,
			},
			{
				InstanceId:     102,
				ReleaseId:      201,
				Artist:         "Pink Floyd",
				Title:          "The Dark Side of the Moon",
				LastUpdatedTime: now,
			},
			{
				InstanceId:     103,
				ReleaseId:      202,
				Artist:         "Miles Davis",
				Title:          "Kind of Blue",
				LastUpdatedTime: now,
			},
		},
	}

	data, err := protov2.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal sample cache: %v", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		t.Fatalf("Failed to write sample cache file: %v", err)
	}

	cm := NewCacheManager(filePath)
	if err := cm.LoadFromDisk(); err != nil {
		t.Fatalf("Expected LoadFromDisk to succeed, got %v", err)
	}

	if cm.GetStatus() != CacheStatusReady {
		t.Errorf("Expected CacheStatusReady, got %v", cm.GetStatus())
	}

	rec101, ok := cm.GetByInstanceID(101)
	if !ok || rec101 == nil {
		t.Fatalf("Expected record with instance ID 101 to be found")
	}
	if rec101.GetTitle() != "The Dark Side of the Moon" {
		t.Errorf("Expected title 'The Dark Side of the Moon', got %q", rec101.GetTitle())
	}

	recs201 := cm.GetByReleaseID(201)
	if len(recs201) != 2 {
		t.Fatalf("Expected 2 records for release ID 201, got %d", len(recs201))
	}

	searchResults := cm.Search("miles")
	if len(searchResults) != 1 {
		t.Fatalf("Expected 1 result for search 'miles', got %d", len(searchResults))
	}
	if searchResults[0].GetArtist() != "Miles Davis" {
		t.Errorf("Expected artist 'Miles Davis', got %q", searchResults[0].GetArtist())
	}
}

func TestCacheLoad_CorruptFileRecovery(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "corrupt_cache.pb")

	// Write invalid corrupt random bytes
	if err := os.WriteFile(filePath, []byte("NOT_A_VALID_PROTOBUF_DATA_CORRUPTED_BYTES"), 0600); err != nil {
		t.Fatalf("Failed to write corrupt file: %v", err)
	}

	cm := NewCacheManager(filePath)
	// Seed with an in-memory record to test that index is cleared on recovery
	cm.Populate([]*pb.Record{createSampleRecord(1, 1, "Artist", "Title")}, time.Now())
	if _, ok := cm.GetByInstanceID(1); !ok {
		t.Fatalf("Expected record 1 to be initially present")
	}

	// Verify LoadFromDisk returns cleanly and recovers without crashing
	if err := cm.LoadFromDisk(); err != nil {
		t.Fatalf("Expected LoadFromDisk() to handle corrupt file cleanly, returned err: %v", err)
	}

	if cm.GetStatus() != CacheStatusRebuilding {
		t.Errorf("Expected status CacheStatusRebuilding, got %v", cm.GetStatus())
	}

	if _, ok := cm.GetByInstanceID(1); ok {
		t.Errorf("Expected in-memory index to be reset, but instance ID 1 was found")
	}

	if len(cm.Search("")) != 0 {
		t.Errorf("Expected search slice to be reset, got %d items", len(cm.Search("")))
	}
}

func TestCacheSave_AtomicPermissions(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "nested", "cache.pb")

	cm := NewCacheManager(filePath)
	records := []*pb.Record{
		createSampleRecord(501, 601, "Radiohead", "OK Computer"),
		createSampleRecord(502, 602, "Radiohead", "Kid A"),
	}
	syncTime := time.Now()
	cm.Populate(records, syncTime)

	if err := cm.SaveToDisk(); err != nil {
		t.Fatalf("SaveToDisk failed: %v", err)
	}

	// Verify target file exists
	fi, err := os.Stat(filePath)
	if err != nil {
		t.Fatalf("Expected cache file to exist: %v", err)
	}

	// Verify strict 0600 permissions
	perm := fi.Mode().Perm()
	if perm != 0600 {
		t.Errorf("Expected file permissions 0600, got %#o", perm)
	}

	// Verify temporary file is cleaned up
	tmpPath := filePath + ".tmp"
	if _, err := os.Stat(tmpPath); !os.IsNotExist(err) {
		t.Errorf("Expected tmp file %s to be cleaned up or renamed", tmpPath)
	}

	// Verify content on disk by loading in fresh manager
	cm2 := NewCacheManager(filePath)
	if err := cm2.LoadFromDisk(); err != nil {
		t.Fatalf("Failed to reload saved file: %v", err)
	}
	if rec, ok := cm2.GetByInstanceID(501); !ok || rec.GetTitle() != "OK Computer" {
		t.Errorf("Expected to reload OK Computer record, got %v", rec)
	}
}

func TestCacheTTL_Expiration(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "ttl_cache.pb")

	cm := NewCacheManager(filePath)

	// Uninitialized or empty cache should be expired
	if !cm.IsExpired(7 * 24 * time.Hour) {
		t.Errorf("Expected uninitialized cache to be expired")
	}

	// Cache populated 8 days ago (expired past default 7-day TTL)
	eightDaysAgo := time.Now().Add(-8 * 24 * time.Hour)
	cm.Populate([]*pb.Record{createSampleRecord(1, 1, "Artist", "Title")}, eightDaysAgo)

	if !cm.IsExpired(0) {
		t.Errorf("Expected 8-day-old cache to be expired with default TTL")
	}
	if !cm.IsExpired(7 * 24 * time.Hour) {
		t.Errorf("Expected 8-day-old cache to be expired with 7-day TTL")
	}

	// Cache populated 1 day ago (not expired past 7-day TTL, but expired past 12-hour TTL)
	oneDayAgo := time.Now().Add(-24 * time.Hour)
	cm.Populate([]*pb.Record{createSampleRecord(1, 1, "Artist", "Title")}, oneDayAgo)

	if cm.IsExpired(0) {
		t.Errorf("Expected 1-day-old cache to NOT be expired with default 7-day TTL")
	}
	if cm.IsExpired(7 * 24 * time.Hour) {
		t.Errorf("Expected 1-day-old cache to NOT be expired with 7-day TTL")
	}
	if !cm.IsExpired(12 * time.Hour) {
		t.Errorf("Expected 1-day-old cache to be expired with 12-hour TTL")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "concurrent_cache.pb")

	cm := NewCacheManager(filePath)
	cm.Populate([]*pb.Record{
		createSampleRecord(1, 10, "Artist 1", "Title 1"),
		createSampleRecord(2, 20, "Artist 2", "Title 2"),
	}, time.Now())

	var wg sync.WaitGroup
	workers := 8
	iterations := 100

	for i := 0; i < workers; i++ {
		workerID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				switch (workerID + j) % 6 {
				case 0:
					cm.Search("Artist")
				case 1:
					cm.GetByInstanceID(int64(j % 10))
				case 2:
					cm.GetByReleaseID(int64(j % 5))
				case 3:
					cm.UpsertRecord(createSampleRecord(int64(j), int64(j%5), fmt.Sprintf("Artist %d", j), fmt.Sprintf("Title %d", j)))
				case 4:
					status := cm.GetStatus()
					cm.SetStatus(status)
				case 5:
					cm.IsExpired(7 * 24 * time.Hour)
				}
			}
		}()
	}

	wg.Wait()
}

func TestConcurrentAccess(t *testing.T) {
	TestCache_ConcurrentAccess(t)
}

