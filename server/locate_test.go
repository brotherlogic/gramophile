package server

import (
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

func TestFormatRecordTitle(t *testing.T) {
	tests := []struct {
		name     string
		record   *pb.Record
		expected string
	}{
		{
			name:     "Nil record",
			record:   nil,
			expected: "",
		},
		{
			name:     "Nil release",
			record:   &pb.Record{},
			expected: "",
		},
		{
			name: "Artist and Title",
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
			record: &pb.Record{
				Release: &pbd.Release{
					Artists: []*pbd.Artist{{Name: "Prince"}},
				},
			},
			expected: "Prince",
		},
		{
			name: "Title only",
			record: &pb.Record{
				Release: &pbd.Release{
					Title: "Untitled Track",
				},
			},
			expected: "Untitled Track",
		},
		{
			name: "Empty artist name and empty title",
			record: &pb.Record{
				Release: &pbd.Release{
					Artists: []*pbd.Artist{{Name: ""}},
					Title:   "",
				},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatRecordTitle(tt.record)
			if got != tt.expected {
				t.Errorf("formatRecordTitle() = %q, expected %q", got, tt.expected)
			}
		})
	}
}

