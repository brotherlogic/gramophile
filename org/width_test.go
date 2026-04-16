package org

import (
	"testing"

	"github.com/brotherlogic/gramophile/db"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestSpillingWithCountDensity(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	for i := 0; i < 10; i++ {
		err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: int64(i + 1), FolderId: 10}})
		if err != nil {
			t.Fatalf("Can't save record: %v", err)
		}
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
		Name: "testing",
		Density: pb.Density_COUNT,
		Foldersets: []*pb.FolderSet{
			{
				Name:   "testing",
				Folder: 10,
				Index:  1,
				Sort:   pb.Sort_LABEL_CATNO,
			}},
		Spaces: []*pb.Space{
			{
				Name:  "Main Shelves",
				Units: 2,
				Width: 4,
			}},
	}, &pb.OrganisationConfig{})

	if err != nil {
		t.Fatalf("Unable to build snapshot: %v", err)
	}

	// 10 records, density COUNT, space width 4, 2 units.
	// Unit 1: 4 records
	// Unit 2: 4 records
	// Spill: 2 records

	counts := make(map[string]map[int32]int)
	for _, p := range snap.GetPlacements() {
		if counts[p.GetSpace()] == nil {
			counts[p.GetSpace()] = make(map[int32]int)
		}
		counts[p.GetSpace()][p.GetUnit()]++
	}

	if counts["Main Shelves"][1] != 4 {
		t.Errorf("Expected 4 records in unit 1 of Main Shelves, got %v", counts["Main Shelves"][1])
	}
	if counts["Main Shelves"][2] != 4 {
		t.Errorf("Expected 4 records in unit 2 of Main Shelves, got %v", counts["Main Shelves"][2])
	}
	if counts["Spill"][1] != 2 {
		t.Errorf("Expected 2 records in unit 1 of Spill, got %v", counts["Spill"][1])
	}
}

func TestFallbackWidthWhenNoWidthsPresent(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	for i := 0; i < 10; i++ {
		err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: int64(i + 1), FolderId: 10}})
		if err != nil {
			t.Fatalf("Can't save record: %v", err)
		}
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
		Name: "testing",
		Density: pb.Density_WIDTH,
		MissingWidthHandling: pb.MissingWidthHandling_MISSING_WIDTH_AVERAGE,
		Foldersets: []*pb.FolderSet{
			{
				Name:   "testing",
				Folder: 10,
				Index:  1,
				Sort:   pb.Sort_LABEL_CATNO,
			}},
		Spaces: []*pb.Space{
			{
				Name:  "Main Shelves",
				Units: 1,
				Width: 5.5, // Should fit 5 records if fallback is 1.0
			}},
	}, &pb.OrganisationConfig{})

	if err != nil {
		t.Fatalf("Unable to build snapshot: %v", err)
	}

	counts := make(map[string]map[int32]int)
	for _, p := range snap.GetPlacements() {
		if counts[p.GetSpace()] == nil {
			counts[p.GetSpace()] = make(map[int32]int)
		}
		counts[p.GetSpace()][p.GetUnit()]++
	}

	if counts["Main Shelves"][1] != 5 {
		t.Errorf("Expected 5 records in Main Shelves, got %v", counts["Main Shelves"][1])
	}
	if counts["Spill"][1] != 5 {
		t.Errorf("Expected 5 records in Spill, got %v", counts["Spill"][1])
	}
}

func TestSpillingWithWidthDensityAndSomeMissingWidths(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	// 2 records with width 2.0
	for i := 0; i < 2; i++ {
		err := d.SaveRecord(ctx, 123, &pb.Record{Width: 2.0, Release: &pbd.Release{InstanceId: int64(i + 1), FolderId: 10}})
		if err != nil {
			t.Fatalf("Can't save record: %v", err)
		}
	}
	// 2 records with no width (should get average 2.0)
	for i := 2; i < 4; i++ {
		err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: int64(i + 1), FolderId: 10}})
		if err != nil {
			t.Fatalf("Can't save record: %v", err)
		}
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
		Name: "testing",
		Density: pb.Density_WIDTH,
		MissingWidthHandling: pb.MissingWidthHandling_MISSING_WIDTH_AVERAGE,
		Foldersets: []*pb.FolderSet{
			{
				Name:   "testing",
				Folder: 10,
				Index:  1,
				Sort:   pb.Sort_LABEL_CATNO,
			}},
		Spaces: []*pb.Space{
			{
				Name:  "Main Shelves",
				Units: 1,
				Width: 5.0, // Should fit 2 records (2*2 = 4, next one would be 4+2=6)
			}},
	}, &pb.OrganisationConfig{})

	if err != nil {
		t.Fatalf("Unable to build snapshot: %v", err)
	}

	counts := make(map[string]map[int32]int)
	for _, p := range snap.GetPlacements() {
		if counts[p.GetSpace()] == nil {
			counts[p.GetSpace()] = make(map[int32]int)
		}
		counts[p.GetSpace()][p.GetUnit()]++
	}

	if counts["Main Shelves"][1] != 2 {
		t.Errorf("Expected 2 records in Main Shelves, got %v", counts["Main Shelves"][1])
	}
	if counts["Spill"][1] != 2 {
		t.Errorf("Expected 2 records in Spill, got %v", counts["Spill"][1])
	}
}
