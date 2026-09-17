package proto

import (
	"testing"

	protov2 "google.golang.org/protobuf/proto"
)

func TestLocationRecord(t *testing.T) {
	loc := &Location{
		Record: "Test Artist - Test Title",
	}

	if loc.GetRecord() != "Test Artist - Test Title" {
		t.Errorf("Expected record 'Test Artist - Test Title', got %v", loc.GetRecord())
	}
}

func TestAdjustSalesProto(t *testing.T) {
	adjust := &AdjustSales{}
	elem := &QueueElement{
		Entry: &QueueElement_AdjustSales{
			AdjustSales: adjust,
		},
	}
	if elem.GetAdjustSales() == nil {
		t.Errorf("Expected non-nil AdjustSales in QueueElement")
	}

	user := &StoredUser{
		LastSaleAdjust: 123456789,
	}
	if user.GetLastSaleAdjust() != 123456789 {
		t.Errorf("Expected LastSaleAdjust to be 123456789, got %v", user.GetLastSaleAdjust())
	}
}

func TestSaleInfoSleeveCondition(t *testing.T) {
	sale := &SaleInfo{
		SleeveCondition: "Mint (M)",
	}
	if sale.GetSleeveCondition() != "Mint (M)" {
		t.Errorf("Expected sleeve condition 'Mint (M)', got %v", sale.GetSleeveCondition())
	}
}

func TestUpdateSaleSleeveCondition(t *testing.T) {
	update := &UpdateSale{
		SleeveCondition: "Near Mint (NM or M-)",
	}
	if update.GetSleeveCondition() != "Near Mint (NM or M-)" {
		t.Errorf("Expected sleeve condition 'Near Mint (NM or M-)', got %v", update.GetSleeveCondition())
	}
}

func TestSyncOrdersProto(t *testing.T) {
	syncOrders := &SyncOrders{
		Page:   1,
		SyncId: 12345,
	}
	elem := &QueueElement{
		Entry: &QueueElement_SyncOrders{
			SyncOrders: syncOrders,
		},
	}
	if elem.GetSyncOrders() == nil {
		t.Errorf("Expected non-nil SyncOrders in QueueElement")
	}
	if elem.GetSyncOrders().GetPage() != 1 {
		t.Errorf("Expected page 1, got %v", elem.GetSyncOrders().GetPage())
	}
	if elem.GetSyncOrders().GetSyncId() != 12345 {
		t.Errorf("Expected sync_id 12345, got %v", elem.GetSyncOrders().GetSyncId())
	}

	user := &StoredUser{
		LastOrderSync: 987654321,
	}
	if user.GetLastOrderSync() != 987654321 {
		t.Errorf("Expected LastOrderSync to be 987654321, got %v", user.GetLastOrderSync())
	}
}

func TestReconcileSalesProto(t *testing.T) {
	reconcileSales := &ReconcileSales{
		Page:      1,
		RefreshId: 12345,
	}
	elem := &QueueElement{
		Entry: &QueueElement_ReconcileSales{
			ReconcileSales: reconcileSales,
		},
	}
	if elem.GetReconcileSales() == nil {
		t.Errorf("Expected non-nil ReconcileSales in QueueElement")
	}
	if elem.GetReconcileSales().GetPage() != 1 {
		t.Errorf("Expected page 1, got %v", elem.GetReconcileSales().GetPage())
	}
	if elem.GetReconcileSales().GetRefreshId() != 12345 {
		t.Errorf("Expected refresh_id 12345, got %v", elem.GetReconcileSales().GetRefreshId())
	}

	user := &StoredUser{
		LastSaleReconcile: 987654321,
	}
	if user.GetLastSaleReconcile() != 987654321 {
		t.Errorf("Expected LastSaleReconcile to be 987654321, got %v", user.GetLastSaleReconcile())
	}
}

func TestGetRecordRequestGetAllRecords(t *testing.T) {
	req := &GetRecordRequest{
		Request: &GetRecordRequest_GetAllRecords{
			GetAllRecords: true,
		},
	}
	if !req.GetGetAllRecords() {
		t.Errorf("Expected GetAllRecords to be true, got %v", req.GetGetAllRecords())
	}
}

func TestRecordCacheProto(t *testing.T) {
	entry1 := &RecordCacheEntry{
		ArtistTitle:       "Artist A - Album A",
		ResolvedTimestamp: 1700000000,
	}
	entry2 := &RecordCacheEntry{
		ArtistTitle:       "Artist B - Album B",
		ResolvedTimestamp: 1700000100,
	}

	if entry1.GetArtistTitle() != "Artist A - Album A" {
		t.Errorf("Expected ArtistTitle 'Artist A - Album A', got %v", entry1.GetArtistTitle())
	}
	if entry1.GetResolvedTimestamp() != 1700000000 {
		t.Errorf("Expected ResolvedTimestamp 1700000000, got %v", entry1.GetResolvedTimestamp())
	}

	cache := &RecordCache{
		InstanceCache: map[int64]*RecordCacheEntry{
			1001: entry1,
		},
		ReleaseCache: map[int64]*RecordCacheEntry{
			2001: entry2,
		},
	}

	if len(cache.GetInstanceCache()) != 1 || cache.GetInstanceCache()[1001].GetArtistTitle() != "Artist A - Album A" {
		t.Errorf("Unexpected instance cache content: %v", cache.GetInstanceCache())
	}
	if len(cache.GetReleaseCache()) != 1 || cache.GetReleaseCache()[2001].GetArtistTitle() != "Artist B - Album B" {
		t.Errorf("Unexpected release cache content: %v", cache.GetReleaseCache())
	}

	data, err := protov2.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal RecordCache: %v", err)
	}

	unmarshaled := &RecordCache{}
	if err := protov2.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal RecordCache: %v", err)
	}

	if unmarshaled.GetInstanceCache()[1001].GetArtistTitle() != "Artist A - Album A" {
		t.Errorf("Unmarshaled instance cache mismatch: %v", unmarshaled.GetInstanceCache())
	}
	if unmarshaled.GetReleaseCache()[2001].GetArtistTitle() != "Artist B - Album B" {
		t.Errorf("Unmarshaled release cache mismatch: %v", unmarshaled.GetReleaseCache())
	}
}

func TestCollectionCacheProto(t *testing.T) {
	record := &CachedRecord{
		InstanceId:      12345,
		Artist:          "The Beatles",
		Title:           "Abbey Road",
		ReleaseId:       67890,
		LastUpdatedTime: 1700000000,
	}

	if record.GetInstanceId() != 12345 {
		t.Errorf("Expected InstanceId 12345, got %v", record.GetInstanceId())
	}
	if record.GetArtist() != "The Beatles" {
		t.Errorf("Expected Artist 'The Beatles', got %v", record.GetArtist())
	}
	if record.GetTitle() != "Abbey Road" {
		t.Errorf("Expected Title 'Abbey Road', got %v", record.GetTitle())
	}
	if record.GetReleaseId() != 67890 {
		t.Errorf("Expected ReleaseId 67890, got %v", record.GetReleaseId())
	}
	if record.GetLastUpdatedTime() != 1700000000 {
		t.Errorf("Expected LastUpdatedTime 1700000000, got %v", record.GetLastUpdatedTime())
	}

	cache := &CollectionCache{
		DiscogsUserId: 99999,
		LastSyncTime:  1700000500,
		Records:       []*CachedRecord{record},
	}

	if cache.GetDiscogsUserId() != 99999 {
		t.Errorf("Expected DiscogsUserId 99999, got %v", cache.GetDiscogsUserId())
	}
	if cache.GetLastSyncTime() != 1700000500 {
		t.Errorf("Expected LastSyncTime 1700000500, got %v", cache.GetLastSyncTime())
	}
	if len(cache.GetRecords()) != 1 {
		t.Fatalf("Expected 1 record, got %d", len(cache.GetRecords()))
	}
	if cache.GetRecords()[0].GetArtist() != "The Beatles" {
		t.Errorf("Expected record artist 'The Beatles', got %v", cache.GetRecords()[0].GetArtist())
	}

	data, err := protov2.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal CollectionCache: %v", err)
	}

	unmarshaled := &CollectionCache{}
	if err := protov2.Unmarshal(data, unmarshaled); err != nil {
		t.Fatalf("Failed to unmarshal CollectionCache: %v", err)
	}

	if unmarshaled.GetDiscogsUserId() != 99999 {
		t.Errorf("Unmarshaled DiscogsUserId mismatch: got %v, expected 99999", unmarshaled.GetDiscogsUserId())
	}
	if unmarshaled.GetLastSyncTime() != 1700000500 {
		t.Errorf("Unmarshaled LastSyncTime mismatch: got %v, expected 1700000500", unmarshaled.GetLastSyncTime())
	}
	if len(unmarshaled.GetRecords()) != 1 {
		t.Fatalf("Unmarshaled records count mismatch: got %d, expected 1", len(unmarshaled.GetRecords()))
	}
	rec := unmarshaled.GetRecords()[0]
	if rec.GetInstanceId() != 12345 || rec.GetArtist() != "The Beatles" || rec.GetTitle() != "Abbey Road" || rec.GetReleaseId() != 67890 || rec.GetLastUpdatedTime() != 1700000000 {
		t.Errorf("Unmarshaled record content mismatch: got %+v", rec)
	}
}

