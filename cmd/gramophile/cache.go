package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	protov2 "google.golang.org/protobuf/proto"
)

// CacheStatus represents the current synchronization/health status of the cache.
type CacheStatus int

const (
	CacheStatusUninitialized CacheStatus = iota
	CacheStatusReady
	CacheStatusSyncing
	CacheStatusStale
	CacheStatusRebuilding
)

func (s CacheStatus) String() string {
	switch s {
	case CacheStatusReady:
		return "Ready"
	case CacheStatusSyncing:
		return "Syncing"
	case CacheStatusStale:
		return "Stale"
	case CacheStatusRebuilding:
		return "Rebuilding"
	default:
		return "Uninitialized"
	}
}

// DefaultCachePath returns the canonical path ~/.gramophile_cache for offline collection storage.
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".gramophile_cache"), nil
}

// CacheManager provides thread-safe local disk persistence, index lookups, and TTL validation.
type CacheManager struct {
	filePath     string
	cache        *pb.CollectionCache
	byInstanceID map[int64]*pb.CachedRecord
	byReleaseID  map[int64][]*pb.CachedRecord
	records      []*pb.CachedRecord
	status       CacheStatus
	mu           sync.RWMutex
}

// NewCacheManager creates a new CacheManager targeting filePath (or ~/.gramophile_cache if empty).
func NewCacheManager(filePath string) *CacheManager {
	if filePath == "" {
		p, err := DefaultCachePath()
		if err == nil {
			filePath = p
		} else {
			filePath = ".gramophile_cache"
		}
	}

	return &CacheManager{
		filePath:     filePath,
		cache:        &pb.CollectionCache{},
		byInstanceID: make(map[int64]*pb.CachedRecord),
		byReleaseID:  make(map[int64][]*pb.CachedRecord),
		status:       CacheStatusUninitialized,
	}
}

func recordToCached(rec *pb.Record) *pb.CachedRecord {
	if rec == nil || rec.GetRelease() == nil {
		return nil
	}
	return &pb.CachedRecord{
		InstanceId:      rec.GetRelease().GetInstanceId(),
		ReleaseId:       rec.GetRelease().GetId(),
		Artist:          getRecordArtist(rec),
		Title:           getRecordTitle(rec),
		LastUpdatedTime: rec.GetLastUpdateTime(),
	}
}

func (cm *CacheManager) resetLocked() {
	cm.cache = &pb.CollectionCache{}
	cm.byInstanceID = make(map[int64]*pb.CachedRecord)
	cm.byReleaseID = make(map[int64][]*pb.CachedRecord)
	cm.records = nil
}

func (cm *CacheManager) rebuildIndexesLocked() {
	cm.byInstanceID = make(map[int64]*pb.CachedRecord)
	cm.byReleaseID = make(map[int64][]*pb.CachedRecord)
	cm.records = make([]*pb.CachedRecord, 0, len(cm.cache.GetRecords()))

	for _, rec := range cm.cache.GetRecords() {
		if rec == nil {
			continue
		}
		cm.records = append(cm.records, rec)
		cm.byInstanceID[rec.GetInstanceId()] = rec
		cm.byReleaseID[rec.GetReleaseId()] = append(cm.byReleaseID[rec.GetReleaseId()], rec)
	}
}

// GetByInstanceID returns the cached record for a specific instance ID.
func (cm *CacheManager) GetByInstanceID(iid int64) (*pb.CachedRecord, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	rec, ok := cm.byInstanceID[iid]
	return rec, ok
}

// GetByReleaseID returns all cached records associated with a release ID.
func (cm *CacheManager) GetByReleaseID(releaseID int64) []*pb.CachedRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	recs := cm.byReleaseID[releaseID]
	if len(recs) == 0 {
		return nil
	}
	res := make([]*pb.CachedRecord, len(recs))
	copy(res, recs)
	return res
}

// GetCachedRecords returns a slice copy of all cached records currently indexed.
func (cm *CacheManager) GetCachedRecords() []*pb.CachedRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	res := make([]*pb.CachedRecord, len(cm.records))
	copy(res, cm.records)
	return res
}

// Search performs a case-insensitive lexical search over artist and title.
func (cm *CacheManager) Search(query string) []*pb.CachedRecord {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	q := strings.TrimSpace(strings.ToLower(query))
	if q == "" {
		res := make([]*pb.CachedRecord, len(cm.records))
		copy(res, cm.records)
		return res
	}

	var matches []*pb.CachedRecord
	for _, rec := range cm.records {
		artist := strings.ToLower(rec.GetArtist())
		title := strings.ToLower(rec.GetTitle())
		combined := artist + " - " + title
		if strings.Contains(artist, q) || strings.Contains(title, q) || strings.Contains(combined, q) {
			matches = append(matches, rec)
		}
	}
	return matches
}

// UpsertRecord updates or adds a single record in the cache and in-memory index maps.
func (cm *CacheManager) UpsertRecord(rec *pb.Record) {
	if rec == nil || rec.GetRelease() == nil {
		return
	}
	cached := recordToCached(rec)
	if cached == nil {
		return
	}

	cm.mu.Lock()
	defer cm.mu.Unlock()

	iid := cached.GetInstanceId()
	oldRec, exists := cm.byInstanceID[iid]

	cm.byInstanceID[iid] = cached

	if exists && oldRec.GetReleaseId() != cached.GetReleaseId() {
		oldList := cm.byReleaseID[oldRec.GetReleaseId()]
		var newList []*pb.CachedRecord
		for _, r := range oldList {
			if r.GetInstanceId() != iid {
				newList = append(newList, r)
			}
		}
		if len(newList) == 0 {
			delete(cm.byReleaseID, oldRec.GetReleaseId())
		} else {
			cm.byReleaseID[oldRec.GetReleaseId()] = newList
		}
	}

	relList := cm.byReleaseID[cached.GetReleaseId()]
	foundInRel := false
	for i, r := range relList {
		if r.GetInstanceId() == iid {
			relList[i] = cached
			foundInRel = true
			break
		}
	}
	if !foundInRel {
		relList = append(relList, cached)
	}
	cm.byReleaseID[cached.GetReleaseId()] = relList

	foundInCache := false
	for i, r := range cm.cache.Records {
		if r.GetInstanceId() == iid {
			cm.cache.Records[i] = cached
			foundInCache = true
			break
		}
	}
	if !foundInCache {
		cm.cache.Records = append(cm.cache.Records, cached)
	}

	foundInSlice := false
	for i, r := range cm.records {
		if r.GetInstanceId() == iid {
			cm.records[i] = cached
			foundInSlice = true
			break
		}
	}
	if !foundInSlice {
		cm.records = append(cm.records, cached)
	}
}

// Populate replaces the cache contents and rebuilds all index mappings.
func (cm *CacheManager) Populate(records []*pb.Record, syncTime time.Time) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.resetLocked()
	cm.cache.LastSyncTime = syncTime.Unix()

	for _, rec := range records {
		cached := recordToCached(rec)
		if cached == nil {
			continue
		}
		iid := cached.GetInstanceId()
		cm.cache.Records = append(cm.cache.Records, cached)
		cm.records = append(cm.records, cached)
		cm.byInstanceID[iid] = cached
		cm.byReleaseID[cached.GetReleaseId()] = append(cm.byReleaseID[cached.GetReleaseId()], cached)
	}

	cm.status = CacheStatusReady
}

// GetStatus returns the current CacheStatus.
func (cm *CacheManager) GetStatus() CacheStatus {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.status
}

// SetStatus updates the CacheStatus.
func (cm *CacheManager) SetStatus(status CacheStatus) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	cm.status = status
}

// SaveToDisk writes binary protobuf to <filePath>.tmp with 0600 permissions and atomically renames to <filePath>.
func (cm *CacheManager) SaveToDisk() error {
	cm.mu.RLock()
	data, err := protov2.Marshal(cm.cache)
	filePath := cm.filePath
	cm.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("failed to marshal collection cache: %w", err)
	}

	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	tmpPath := filePath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary cache file: %w", err)
	}

	if err := os.Chmod(tmpPath, 0600); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to chmod temporary cache file: %w", err)
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("failed to atomically rename cache file: %w", err)
	}

	return nil
}

// LoadFromDisk reads binary protobuf from disk. Handles missing files cleanly, and on corrupt/truncated bytes resets index, transitions status to CacheStatusRebuilding, and returns cleanly.
func (cm *CacheManager) LoadFromDisk() error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	data, err := os.ReadFile(cm.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			cm.resetLocked()
			cm.status = CacheStatusUninitialized
			return nil
		}
		return err
	}

	var cache pb.CollectionCache
	if err := protov2.Unmarshal(data, &cache); err != nil {
		cm.resetLocked()
		cm.status = CacheStatusRebuilding
		return nil
	}

	cm.cache = &cache
	cm.rebuildIndexesLocked()
	cm.status = CacheStatusReady
	return nil
}

// IsExpired checks whether time.Since(time.Unix(cache.GetLastSyncTime(), 0)) > ttl (default 7 days).
func (cm *CacheManager) IsExpired(ttl time.Duration) bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if ttl <= 0 {
		ttl = 7 * 24 * time.Hour
	}

	if cm.cache == nil || cm.cache.GetLastSyncTime() <= 0 {
		return true
	}

	return time.Since(time.Unix(cm.cache.GetLastSyncTime(), 0)) > ttl
}
