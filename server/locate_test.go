package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestLocateRecord_Success(t *testing.T) {
	ctx := getTestContext(123)
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{Name: "test-org"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("cannot save user: %v", err)
	}

	s := Server{d: d}

	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         100,
			InstanceId: 1001,
			Title:      "Album100",
			Artists:    []*pbd.Artist{{Name: "ArtistA"}},
		},
	})
	if err != nil {
		t.Fatalf("cannot save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         101,
			InstanceId: 1002,
			Title:      "Album101",
			Artists:    []*pbd.Artist{{Name: "ArtistB"}},
		},
	})
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         100,
			InstanceId: 1003,
			Title:      "Album100_2",
		},
	})

	err = d.SaveSnapshot(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
	}, "test-org", &pb.OrganisationSnapshot{
		Date: 12345,
		Placements: []*pb.Placement{
			{Iid: 1002, Space: "ShelfA", Unit: 1, Index: 1},
			{Iid: 1001, Space: "ShelfA", Unit: 1, Index: 2},
			{Iid: 1003, Space: "ShelfA", Unit: 1, Index: 3},
		},
	})
	if err != nil {
		t.Fatalf("cannot save snapshot: %v", err)
	}

	res, err := s.LocateRecord(ctx, &pb.LocateRecordRequest{
		ReleaseId: 100,
	})

	if err != nil {
		t.Fatalf("LocateRecord returned error: %v", err)
	}

	if len(res.GetLocations()) != 2 {
		t.Fatalf("Expected 2 locations, got %v", len(res.GetLocations()))
	}

	// Check first location (1001) which is at index 2 (middle)
	loc1 := res.GetLocations()[0]
	if loc1.GetShelf() != "ShelfA" || loc1.GetSlot() != 1 {
		t.Errorf("Bad location 1: %v", loc1)
	}
	if loc1.GetRecord() != "ArtistA - Album100" {
		t.Errorf("Expected target record 'ArtistA - Album100', got '%v'", loc1.GetRecord())
	}
	if len(loc1.GetBefore()) != 1 || loc1.GetBefore()[0].GetIid() != 1002 {
		t.Errorf("Bad before context 1: %v", loc1.GetBefore())
	}
	if loc1.GetBefore()[0].GetRecord() != "ArtistB - Album101" {
		t.Errorf("Expected before context record 'ArtistB - Album101', got '%v'", loc1.GetBefore()[0].GetRecord())
	}
	if len(loc1.GetAfter()) != 1 || loc1.GetAfter()[0].GetIid() != 1003 {
		t.Errorf("Bad after context 1: %v", loc1.GetAfter())
	}
	if loc1.GetAfter()[0].GetRecord() != "Album100_2" {
		t.Errorf("Expected after context record 'Album100_2', got '%v'", loc1.GetAfter()[0].GetRecord())
	}
}

func TestLocateRecord_NotFound(t *testing.T) {
	ctx := getTestContext(123)
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("cannot save user: %v", err)
	}

	s := Server{d: d}

	_, err = s.LocateRecord(ctx, &pb.LocateRecordRequest{
		ReleaseId: 999,
	})

	if err == nil {
		t.Fatalf("Expected error for non-existent release, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.NotFound {
		t.Errorf("Expected NotFound error code, got %v (err: %v)", st.Code(), err)
	}
}

func TestLocateRecord_NotInOrg(t *testing.T) {
	ctx := getTestContext(123)
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{Name: "test-org"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("cannot save user: %v", err)
	}

	s := Server{d: d}

	// Save record in collection
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			Id:         200,
			InstanceId: 2001,
			Title:      "Unplaced Album",
			Artists:    []*pbd.Artist{{Name: "ArtistC"}},
		},
	})
	if err != nil {
		t.Fatalf("cannot save record: %v", err)
	}

	// Save snapshot without record 2001
	err = d.SaveSnapshot(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
	}, "test-org", &pb.OrganisationSnapshot{
		Date: 12345,
		Placements: []*pb.Placement{
			{Iid: 9999, Space: "ShelfA", Unit: 1, Index: 1},
		},
	})
	if err != nil {
		t.Fatalf("cannot save snapshot: %v", err)
	}

	_, err = s.LocateRecord(ctx, &pb.LocateRecordRequest{
		ReleaseId: 200,
	})

	if err == nil {
		t.Fatalf("Expected error for unplaced record, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.InvalidArgument {
		t.Errorf("Expected InvalidArgument error code, got %v (err: %v)", st.Code(), err)
	}
	if st.Message() != "record not in any organization" {
		t.Errorf("Expected error message 'record not in any organization', got %q", st.Message())
	}
}

func TestLocateRecord_Unauthenticated(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	s := Server{d: d}

	_, err := s.LocateRecord(context.Background(), &pb.LocateRecordRequest{
		ReleaseId: 100,
	})

	if err == nil {
		t.Fatalf("Expected error for unauthenticated request, got nil")
	}
	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.Unauthenticated {
		t.Errorf("Expected Unauthenticated error code, got %v (err: %v)", st.Code(), err)
	}
}

func TestLocateRecord_ContextBoundaries(t *testing.T) {
	ctx := getTestContext(123)
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{Name: "multi-org"},
					{Name: "single-org"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("cannot save user: %v", err)
	}

	s := Server{d: d}

	// Create 5 records for multi-org
	records := []*pb.Record{
		{Release: &pbd.Release{Id: 101, InstanceId: 1001, Title: "Rec1", Artists: []*pbd.Artist{{Name: "Artist1"}}}},
		{Release: &pbd.Release{Id: 102, InstanceId: 1002, Title: "Rec2", Artists: []*pbd.Artist{{Name: "Artist2"}}}},
		{Release: &pbd.Release{Id: 103, InstanceId: 1003, Title: "Rec3", Artists: []*pbd.Artist{{Name: "Artist3"}}}},
		{Release: &pbd.Release{Id: 104, InstanceId: 1004, Title: "Rec4", Artists: []*pbd.Artist{{Name: "Artist4"}}}},
		{Release: &pbd.Release{Id: 105, InstanceId: 1005, Title: "Rec5", Artists: []*pbd.Artist{{Name: "Artist5"}}}},
		{Release: &pbd.Release{Id: 201, InstanceId: 2001, Title: "SingleRec", Artists: []*pbd.Artist{{Name: "ArtistSolo"}}}},
	}
	for _, rec := range records {
		if err := d.SaveRecord(ctx, 123, rec); err != nil {
			t.Fatalf("cannot save record %v: %v", rec.GetRelease().GetInstanceId(), err)
		}
	}

	// Save snapshot with 5 placements
	err = d.SaveSnapshot(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
	}, "multi-org", &pb.OrganisationSnapshot{
		Date: 12345,
		Placements: []*pb.Placement{
			{Iid: 1001, Space: "ShelfA", Unit: 1, Index: 1},
			{Iid: 1002, Space: "ShelfA", Unit: 1, Index: 2},
			{Iid: 1003, Space: "ShelfA", Unit: 1, Index: 3},
			{Iid: 1004, Space: "ShelfA", Unit: 1, Index: 4},
			{Iid: 1005, Space: "ShelfA", Unit: 1, Index: 5},
		},
	})
	if err != nil {
		t.Fatalf("cannot save multi-org snapshot: %v", err)
	}

	// Save snapshot with 1 placement
	err = d.SaveSnapshot(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
	}, "single-org", &pb.OrganisationSnapshot{
		Date: 12345,
		Placements: []*pb.Placement{
			{Iid: 2001, Space: "ShelfB", Unit: 2, Index: 1},
		},
	})
	if err != nil {
		t.Fatalf("cannot save single-org snapshot: %v", err)
	}

	// 1. Boundary: Record at start of shelf (index 0 / 1001) -> 0 before items, 2 after items
	resStart, err := s.LocateRecord(ctx, &pb.LocateRecordRequest{ReleaseId: 101})
	if err != nil {
		t.Fatalf("LocateRecord(101) returned error: %v", err)
	}
	if len(resStart.GetLocations()) != 1 {
		t.Fatalf("Expected 1 location for 101, got %v", len(resStart.GetLocations()))
	}
	locStart := resStart.GetLocations()[0]
	if len(locStart.GetBefore()) != 0 {
		t.Errorf("Expected 0 before items at start of shelf, got %v", len(locStart.GetBefore()))
	}
	if len(locStart.GetAfter()) != 2 {
		t.Errorf("Expected 2 after items at start of shelf, got %v", len(locStart.GetAfter()))
	} else {
		if locStart.GetAfter()[0].GetIid() != 1002 || locStart.GetAfter()[1].GetIid() != 1003 {
			t.Errorf("Unexpected after context: %v", locStart.GetAfter())
		}
	}

	// 2. Boundary: Record at end of shelf (index 4 / 1005) -> 2 before items, 0 after items
	resEnd, err := s.LocateRecord(ctx, &pb.LocateRecordRequest{ReleaseId: 105})
	if err != nil {
		t.Fatalf("LocateRecord(105) returned error: %v", err)
	}
	if len(resEnd.GetLocations()) != 1 {
		t.Fatalf("Expected 1 location for 105, got %v", len(resEnd.GetLocations()))
	}
	locEnd := resEnd.GetLocations()[0]
	if len(locEnd.GetBefore()) != 2 {
		t.Errorf("Expected 2 before items at end of shelf, got %v", len(locEnd.GetBefore()))
	} else {
		if locEnd.GetBefore()[0].GetIid() != 1004 || locEnd.GetBefore()[1].GetIid() != 1003 {
			t.Errorf("Unexpected before context: %v", locEnd.GetBefore())
		}
	}
	if len(locEnd.GetAfter()) != 0 {
		t.Errorf("Expected 0 after items at end of shelf, got %v", len(locEnd.GetAfter()))
	}

	// 3. Boundary: Record in middle of shelf (index 2 / 1003) -> 2 before items, 2 after items
	resMid, err := s.LocateRecord(ctx, &pb.LocateRecordRequest{ReleaseId: 103})
	if err != nil {
		t.Fatalf("LocateRecord(103) returned error: %v", err)
	}
	if len(resMid.GetLocations()) != 1 {
		t.Fatalf("Expected 1 location for 103, got %v", len(resMid.GetLocations()))
	}
	locMid := resMid.GetLocations()[0]
	if len(locMid.GetBefore()) != 2 || len(locMid.GetAfter()) != 2 {
		t.Errorf("Expected 2 before and 2 after items in middle of shelf, got before=%v after=%v", len(locMid.GetBefore()), len(locMid.GetAfter()))
	}

	// 4. Boundary: Single record on shelf (index 0 / 2001) -> 0 before items, 0 after items
	resSingle, err := s.LocateRecord(ctx, &pb.LocateRecordRequest{ReleaseId: 201})
	if err != nil {
		t.Fatalf("LocateRecord(201) returned error: %v", err)
	}
	if len(resSingle.GetLocations()) != 1 {
		t.Fatalf("Expected 1 location for 201, got %v", len(resSingle.GetLocations()))
	}
	locSingle := resSingle.GetLocations()[0]
	if len(locSingle.GetBefore()) != 0 {
		t.Errorf("Expected 0 before items for single record, got %v", len(locSingle.GetBefore()))
	}
	if len(locSingle.GetAfter()) != 0 {
		t.Errorf("Expected 0 after items for single record, got %v", len(locSingle.GetAfter()))
	}
}

func TestFormatRecordTitle(t *testing.T) {
	tests := []struct {
		name     string
		iid      int64
		record   *pb.Record
		expected string
	}{
		{
			name:     "Nil record",
			iid:      1001,
			record:   nil,
			expected: "Unknown (1001)",
		},
		{
			name:     "Nil release",
			iid:      1002,
			record:   &pb.Record{},
			expected: "Unknown (1002)",
		},
		{
			name: "Artist and Title",
			iid:  1003,
			record: &pb.Record{
				Release: &pbd.Release{
					Title:   "Thriller",
					Artists: []*pbd.Artist{{Name: "Michael Jackson"}},
				},
			},
			expected: "Michael Jackson - Thriller",
		},
		{
			name: "Artist only",
			iid:  1004,
			record: &pb.Record{
				Release: &pbd.Release{
					Artists: []*pbd.Artist{{Name: "Prince"}},
				},
			},
			expected: "Prince",
		},
		{
			name: "Title only",
			iid:  1005,
			record: &pb.Record{
				Release: &pbd.Release{
					Title: "Untitled Track",
				},
			},
			expected: "Untitled Track",
		},
		{
			name: "Empty artist name and empty title",
			iid:  1006,
			record: &pb.Record{
				Release: &pbd.Release{
					Artists: []*pbd.Artist{{Name: ""}},
					Title:   "",
				},
			},
			expected: "Unknown (1006)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRecordTitle(tt.iid, tt.record)
			if got != tt.expected {
				t.Errorf("formatRecordTitle() = %q, expected %q", got, tt.expected)
			}
		})
	}
}
