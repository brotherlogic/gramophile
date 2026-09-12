package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"golang.org/x/sync/singleflight"
	"google.golang.org/protobuf/proto"
)

const (
	defaultCacheTTL = 7 * 24 * time.Hour
	workerPoolLimit = 8
)

// RecordCacheManager manages a thread-safe in-memory and persistent on-disk
// cache of record instance and release artist-title strings.
type RecordCacheManager struct {
	mu            sync.RWMutex
	saveMu        sync.Mutex
	data          *pb.RecordCache
	filePath      string
	sfg           singleflight.Group
	workerSem     chan struct{}
	retryBackoffs []time.Duration
}

// NewRecordCacheManager initializes a new RecordCacheManager.
// If customPath is provided, it uses that path; otherwise it defaults to ~/.gramophile_cache.
// If the file is missing, it initializes an empty cache.
// If the file is corrupted, it emits a warning to stderr, resets an empty cache, and continues.
func NewRecordCacheManager(customPath ...string) (*RecordCacheManager, error) {
	var filePath string
	if len(customPath) > 0 && customPath[0] != "" {
		filePath = customPath[0]
	} else {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("unable to resolve user home directory: %w", err)
		}
		filePath = filepath.Join(homeDir, ".gramophile_cache")
	}

	if strings.HasPrefix(filePath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			filePath = filepath.Join(homeDir, filePath[2:])
		}
	}

	mgr := &RecordCacheManager{
		filePath:      filePath,
		workerSem:     make(chan struct{}, workerPoolLimit),
		retryBackoffs: []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second},
		data: &pb.RecordCache{
			InstanceCache: make(map[int64]*pb.RecordCacheEntry),
			ReleaseCache:  make(map[int64]*pb.RecordCacheEntry),
		},
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return mgr, nil
		}
		return nil, fmt.Errorf("unable to read cache file %v: %w", filePath, err)
	}

	cache := &pb.RecordCache{}
	if err := proto.Unmarshal(content, cache); err != nil {
		fmt.Fprintf(os.Stderr, "warning: corrupted record cache at %v: %v; resetting empty cache\n", filePath, err)
		return mgr, nil
	}

	if cache.InstanceCache == nil {
		cache.InstanceCache = make(map[int64]*pb.RecordCacheEntry)
	}
	if cache.ReleaseCache == nil {
		cache.ReleaseCache = make(map[int64]*pb.RecordCacheEntry)
	}
	mgr.data = cache

	return mgr, nil
}

// Save marshals RecordCache into binary proto, writes to a temporary file
// with 0600 permissions, and atomically renames it to the target file path.
func (m *RecordCacheManager) Save() error {
	m.saveMu.Lock()
	defer m.saveMu.Unlock()

	m.mu.RLock()
	data, err := proto.Marshal(m.data)
	m.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("unable to marshal record cache: %w", err)
	}

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("unable to create directory %v: %w", dir, err)
	}

	tmpFile, err := os.CreateTemp(dir, ".gramophile_cache_tmp_*")
	if err != nil {
		return fmt.Errorf("unable to create temp cache file: %w", err)
	}
	tmpName := tmpFile.Name()
	defer func() {
		if tmpFile != nil {
			tmpFile.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmpFile.Chmod(0600); err != nil {
		return fmt.Errorf("unable to chmod temp cache file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("unable to write temp cache file: %w", err)
	}

	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("unable to sync temp cache file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("unable to close temp cache file: %w", err)
	}
	tmpFile = nil

	if err := os.Rename(tmpName, m.filePath); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("unable to rename temp file to %v: %w", m.filePath, err)
	}

	return nil
}

func formatArtistTitle(r *pbd.Release) string {
	if r == nil {
		return ""
	}
	return fmt.Sprintf("%v - %v", getArtist(r), r.GetTitle())
}

// ResolveInstance resolves the artist-title for an instance ID.
// Fresh hit (<= 7 days): returns cached value immediately.
// Stale hit (> 7 days): returns cached value immediately and triggers background refresh.
// Cache miss: fetches via singleflight with retry loop, saves to cache, and returns formatted Artist - Title.
// Fallback: returns [Unresolved: <iid>] on persistent failure.
func (m *RecordCacheManager) ResolveInstance(ctx context.Context, client pb.GramophileEServiceClient, iid int64) (string, error) {
	m.mu.RLock()
	entry, exists := m.data.GetInstanceCache()[iid]
	if exists && entry != nil {
		age := time.Since(time.Unix(entry.GetResolvedTimestamp(), 0))
		val := entry.GetArtistTitle()
		m.mu.RUnlock()

		if age <= defaultCacheTTL {
			return val, nil
		}

		// Stale hit: return immediately, dispatch background refresh
		go func() {
			_, _ = m.refreshInstance(context.Background(), client, iid)
		}()
		return val, nil
	}
	m.mu.RUnlock()

	return m.refreshInstance(ctx, client, iid)
}

func (m *RecordCacheManager) refreshInstance(ctx context.Context, client pb.GramophileEServiceClient, iid int64) (string, error) {
	key := fmt.Sprintf("iid:%d", iid)
	val, err, _ := m.sfg.Do(key, func() (interface{}, error) {
		m.mu.RLock()
		if entry, exists := m.data.GetInstanceCache()[iid]; exists && entry != nil {
			age := time.Since(time.Unix(entry.GetResolvedTimestamp(), 0))
			if age <= defaultCacheTTL {
				artistTitle := entry.GetArtistTitle()
				m.mu.RUnlock()
				return artistTitle, nil
			}
		}
		m.mu.RUnlock()

		select {
		case m.workerSem <- struct{}{}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		defer func() { <-m.workerSem }()

		req := &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					InstanceId: iid,
				},
			},
		}

		backoffs := m.retryBackoffs
		if len(backoffs) == 0 {
			backoffs = []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
		}

		var lastErr error
		var resp *pb.GetRecordResponse
		for attempt := 0; attempt < len(backoffs); attempt++ {
			resp, lastErr = client.GetRecord(ctx, req)
			if lastErr == nil && resp != nil && len(resp.GetRecords()) > 0 && resp.GetRecords()[0].GetRecord() != nil && resp.GetRecords()[0].GetRecord().GetRelease() != nil {
				break
			}
			if attempt < len(backoffs)-1 {
				select {
				case <-time.After(backoffs[attempt]):
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
		}

		if lastErr != nil || resp == nil || len(resp.GetRecords()) == 0 || resp.GetRecords()[0].GetRecord() == nil || resp.GetRecords()[0].GetRecord().GetRelease() == nil {
			return fmt.Sprintf("[Unresolved: %d]", iid), nil
		}

		release := resp.GetRecords()[0].GetRecord().GetRelease()
		artistTitle := formatArtistTitle(release)

		m.mu.Lock()
		m.data.GetInstanceCache()[iid] = &pb.RecordCacheEntry{
			ArtistTitle:       artistTitle,
			ResolvedTimestamp: time.Now().Unix(),
		}
		m.mu.Unlock()

		_ = m.Save()
		return artistTitle, nil
	})

	if err != nil {
		return "", err
	}
	return val.(string), nil
}

// ResolveRelease resolves the artist-title for a release ID.
// Fresh hit (<= 7 days): returns cached value immediately.
// Stale hit (> 7 days): returns cached value immediately and triggers background refresh.
// Cache miss: fetches via singleflight with retry loop, saves to cache, and returns formatted Artist - Title.
// Fallback: returns [Unresolved: <releaseId>] on persistent failure.
func (m *RecordCacheManager) ResolveRelease(ctx context.Context, client pb.GramophileEServiceClient, releaseId int64) (string, error) {
	m.mu.RLock()
	entry, exists := m.data.GetReleaseCache()[releaseId]
	if exists && entry != nil {
		age := time.Since(time.Unix(entry.GetResolvedTimestamp(), 0))
		val := entry.GetArtistTitle()
		m.mu.RUnlock()

		if age <= defaultCacheTTL {
			return val, nil
		}

		// Stale hit: return immediately, dispatch background refresh
		go func() {
			_, _ = m.refreshRelease(context.Background(), client, releaseId)
		}()
		return val, nil
	}
	m.mu.RUnlock()

	return m.refreshRelease(ctx, client, releaseId)
}

func (m *RecordCacheManager) refreshRelease(ctx context.Context, client pb.GramophileEServiceClient, releaseId int64) (string, error) {
	key := fmt.Sprintf("release:%d", releaseId)
	val, err, _ := m.sfg.Do(key, func() (interface{}, error) {
		m.mu.RLock()
		if entry, exists := m.data.GetReleaseCache()[releaseId]; exists && entry != nil {
			age := time.Since(time.Unix(entry.GetResolvedTimestamp(), 0))
			if age <= defaultCacheTTL {
				artistTitle := entry.GetArtistTitle()
				m.mu.RUnlock()
				return artistTitle, nil
			}
		}
		m.mu.RUnlock()

		select {
		case m.workerSem <- struct{}{}:
		case <-ctx.Done():
			return "", ctx.Err()
		}
		defer func() { <-m.workerSem }()

		req := &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					ReleaseId: releaseId,
				},
			},
		}

		backoffs := m.retryBackoffs
		if len(backoffs) == 0 {
			backoffs = []time.Duration{500 * time.Millisecond, 1 * time.Second, 2 * time.Second}
		}

		var lastErr error
		var resp *pb.GetRecordResponse
		for attempt := 0; attempt < len(backoffs); attempt++ {
			resp, lastErr = client.GetRecord(ctx, req)
			if lastErr == nil && resp != nil && len(resp.GetRecords()) > 0 && resp.GetRecords()[0].GetRecord() != nil && resp.GetRecords()[0].GetRecord().GetRelease() != nil {
				break
			}
			if attempt < len(backoffs)-1 {
				select {
				case <-time.After(backoffs[attempt]):
				case <-ctx.Done():
					return "", ctx.Err()
				}
			}
		}

		if lastErr != nil || resp == nil || len(resp.GetRecords()) == 0 || resp.GetRecords()[0].GetRecord() == nil || resp.GetRecords()[0].GetRecord().GetRelease() == nil {
			return fmt.Sprintf("[Unresolved: %d]", releaseId), nil
		}

		release := resp.GetRecords()[0].GetRecord().GetRelease()
		artistTitle := formatArtistTitle(release)

		m.mu.Lock()
		m.data.GetReleaseCache()[releaseId] = &pb.RecordCacheEntry{
			ArtistTitle:       artistTitle,
			ResolvedTimestamp: time.Now().Unix(),
		}
		m.mu.Unlock()

		_ = m.Save()
		return artistTitle, nil
	})

	if err != nil {
		return "", err
	}
	return val.(string), nil
}
