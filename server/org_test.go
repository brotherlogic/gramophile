package server

import (
	"log"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"
	"google.golang.org/protobuf/proto"
)

func abs(a float32) float32 {
	if a < 0 {
		return 0 - a
	}
	return a
}

func TestLabelOrdering(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && o.Iid != 1234 {
			bp = true
		}
		if o.Index == 2 && o.Iid != 1236 {
			bp = true
		}
		if o.Index == 3 && o.Iid != 1235 {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement")
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v", o.Index, o.Iid)
		}
	}
}

func TestReleaseDateOrdering(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, ReleaseDate: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, ReleaseDate: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, ReleaseDate: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_RELEASE_YEAR,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && o.Iid != 1234 {
			bp = true
		}
		if o.Index == 2 && o.Iid != 1235 {
			bp = true
		}
		if o.Index == 3 && o.Iid != 1236 {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement")
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v", o.Index, o.Iid)
		}
	}
}

func TestReleaseDateOrdering_IgnoresGrouping(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{Width: 5, Release: &pbd.Release{InstanceId: 1234, ReleaseDate: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Width: 5, Release: &pbd.Release{InstanceId: 1235, ReleaseDate: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Width: 5, Release: &pbd.Release{InstanceId: 1236, ReleaseDate: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:    "testing",
						Density: pb.Density_WIDTH,
						Grouping: &pb.Grouping{
							Type: pb.GroupingType_GROUPING_GROUP,
						},
						Spill: &pb.Spill{
							Type:      pb.GroupSpill_SPILL_BREAK_ORDERING,
							LookAhead: -1,
						},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_RELEASE_YEAR,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 10,
							},
							{
								Name:  "Second Shelves",
								Units: 1,
								Width: 100,
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.GetSpace() == "Second Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.GetSpace() == "Second Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.GetSpace() == "Second Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement")
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_NoGroupingNoSpill(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1237, FolderId: 12, Labels: []*pbd.Label{{Name: "CC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_NO_GROUPING},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_NO_SPILL},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot:(%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Second Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestReleaseYearOrdering_NoGroupingNoSpill(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1234, ReleaseDate: 10, FolderId: 12, Labels: []*pbd.Label{{Name: "DDD"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, ReleaseDate: 11, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1237, ReleaseDate: 13, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, ReleaseDate: 12, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_NO_GROUPING},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_NO_SPILL},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_RELEASE_YEAR,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot:(%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestReleaseEarliestYearOrdering_NoGroupingNoSpill(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:               10,
		Sleeve:              "Madeup",
		EarliestReleaseDate: 10,
		Release:             &pbd.Release{InstanceId: 1234, ReleaseDate: 13, FolderId: 12, Labels: []*pbd.Label{{Name: "DDD"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:               50,
		Sleeve:              "Madeup",
		EarliestReleaseDate: 12,
		Release:             &pbd.Release{InstanceId: 1235, ReleaseDate: 12, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:               50,
		Sleeve:              "Madeup",
		EarliestReleaseDate: 14,
		Release:             &pbd.Release{InstanceId: 1237, ReleaseDate: 11, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:               10,
		Sleeve:              "Madeup",
		EarliestReleaseDate: 14,
		Release:             &pbd.Release{InstanceId: 1236, ReleaseDate: 10, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_NO_GROUPING},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_NO_SPILL},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_EARLIEST_RELEASE_YEAR,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot:(%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_NoGroupingInfiniteSpill(t *testing.T) {
	log.Printf("RUNNING")
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1237, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "CC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_NO_GROUPING},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_BREAK_ORDERING, LookAhead: -1},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot: (%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_GroupingFail(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_GROUP},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_NO_SPILL},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  90,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  90,
								Layout: pb.Layout_TIGHT,
							},
						},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 2 {
		t.Fatalf("Missing record in snapshot: (%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1236 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_GroupingNoSpill(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1237, FolderId: 12, Labels: []*pbd.Label{{Name: "CC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_GROUP},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_NO_SPILL},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Third Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot: (%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Second Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Second Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Third Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_GroupingAndSpill(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{
		{Id: 5, Name: "Width"},
		{Id: 10, Name: "Sleeve"},
	}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "1"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)

	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   50,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1237, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB", Catno: "2"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   10,
		Sleeve:  "Madeup",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
			SleeveConfig: &pb.SleeveConfig{
				Mandate:        pb.Mandate_REQUIRED,
				AllowedSleeves: []*pb.Sleeve{{Name: "Madeup", WidthMultiplier: 1.0}},
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:     "testing",
						Density:  pb.Density_WIDTH,
						Grouping: &pb.Grouping{Type: pb.GroupingType_GROUPING_GROUP},
						Spill:    &pb.Spill{Type: pb.GroupSpill_SPILL_BREAK_ORDERING, LookAhead: -1},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Second Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							},
							{
								Name:   "Third Main Shelves",
								Units:  1,
								Width:  100,
								Layout: pb.Layout_TIGHT,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 4 {
		t.Fatalf("Missing record in snapshot: (%v) %v", len(org.GetSnapshot().GetPlacements()), org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(100+10+10)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth)
	}

	bp := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Space != "Main Shelves") {
			bp = true
		}
		if o.Index == 3 && (o.Iid != 1236 || o.Space != "Second Main Shelves") {
			bp = true
		}
		if o.Index == 4 && (o.Iid != 1237 || o.Space != "Second Main Shelves") {
			bp = true
		}
	}

	if bp {
		t.Errorf("Bad placement (%v)", totalWidth)
		for _, o := range org.Snapshot.GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o)
		}
	}
}

func TestLabelOrdering_WithOverrides(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Id: 1, Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Id: 3, Name: "CCC"}, {Id: 2, Name: "BBB", Catno: "First"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Id: 2, Name: "BBB", Catno: "Second"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Grouping: &pb.Grouping{
							Type: pb.GroupingType_GROUPING_NO_GROUPING,
							LabelWeights: []*pb.LabelWeight{
								{
									Weight:  0.8,
									LabelId: 2,
								},
							},
						},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	bad := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && o.Iid != 1234 {
			bad = true
		}
		if o.Index == 2 && o.Iid != 1235 {
			bad = true
		}
		if o.Index == 3 && o.Iid != 1236 {
			bad = true
		}
	}

	if bad {
		t.Errorf("Bad placemen")
		for _, o := range org.GetSnapshot().GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o.SortKey)
		}
	}
}

func TestArtistOrdering_WithOverrides(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Id: 1, Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Id: 3, Name: "CCC"}, {Id: 2, Name: "BBB", Catno: "First"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Id: 2, Name: "BBB", Catno: "Second"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Grouping: &pb.Grouping{
							Type: pb.GroupingType_GROUPING_NO_GROUPING,
							LabelWeights: []*pb.LabelWeight{
								{
									Weight:  0.8,
									LabelId: 2,
								},
							},
						},
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	bad := false
	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && o.Iid != 1234 {
			bad = true
		}
		if o.Index == 2 && o.Iid != 1235 {
			bad = true
		}
		if o.Index == 3 && o.Iid != 1236 {
			bad = true
		}
	}

	if bad {
		t.Errorf("Bad placemen")
		for _, o := range org.GetSnapshot().GetPlacements() {
			t.Errorf("%v. %v -> %v", o.Index, o.Iid, o.SortKey)
		}
	}
}

func TestWidths(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Width"}, {Id: 5, Name: "Sleeve"}}}
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Width:   2.4,
		Sleeve:  "TestSleeve",
		Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Width:   2.5,
		Sleeve:  "TestSleeve",
		Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			SleeveConfig: &pb.SleeveConfig{
				AllowedSleeves: []*pb.Sleeve{{Name: "TestSleeve", WidthMultiplier: 1.5}},
				Mandate:        pb.Mandate_REQUIRED},
			WidthConfig: &pb.WidthConfig{
				Mandate: pb.Mandate_REQUIRED,
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:    "testing",
						Density: pb.Density_WIDTH,
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:   "Main Shelves",
								Units:  2,
								Width:  100,
								Layout: pb.Layout_LOOSE,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 2 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	totalWidth := float32(0)
	for _, o := range org.GetSnapshot().GetPlacements() {
		totalWidth += o.GetWidth()
	}
	if abs(totalWidth-(2.4*1.5+2.5*1.5)) > 0.01 {
		t.Errorf("Wrong width returned: %v", totalWidth-2.4*1.5+2.5*1.5)
	}
}

func TestGetSnapshotHash(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Missing record in snapshot: %v", org)
	}

	time.Sleep(time.Second * 2)

	org2, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get second org version")
	}

	if org.GetSnapshot().GetDate() == org2.GetSnapshot().GetDate() || org.GetSnapshot().GetHash() != org2.GetSnapshot().GetHash() {
		t.Errorf("Hash or Date mismatch on second pull: %v vs %v", org.GetSnapshot(), org2.GetSnapshot())
		for i, placement := range org.GetSnapshot().Placements {
			t.Errorf("S1: %v", placement)
			t.Errorf("S2: %v", org.GetSnapshot().Placements[i])
		}
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1237, FolderId: 12, Labels: []*pbd.Label{{Name: "DDD"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	time.Sleep(time.Second * 2)

	org3, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get second org version")
	}

	if org.GetSnapshot().GetDate() == org3.GetSnapshot().GetDate() || org.GetSnapshot().GetHash() == org3.GetSnapshot().GetHash() {
		t.Errorf("Hash or Date mismatch on third pull: %v vs %v", org.GetSnapshot(), org2.GetSnapshot())
	}
}

func applyMoves(snapshot *pb.OrganisationSnapshot, moves []*pb.Move) *pb.OrganisationSnapshot {
	// Copy orginal placements and ensure that they're sorted
	placements := make([]*pb.Placement, len(snapshot.GetPlacements()))
	for _, p := range snapshot.GetPlacements() {
		placements = append(placements, proto.Clone(p).(*pb.Placement))
	}
	indexToRecord := make(map[int32]*pb.Placement)
	for _, placement := range placements {
		indexToRecord[placement.GetIndex()] = placement
	}

	for _, m := range moves {
		if m.GetStart().GetIndex() != m.GetEnd().GetIndex() {
			nIndex := make(map[int32]*pb.Placement)
			found := indexToRecord[m.GetStart().GetIndex()]
			if m.GetStart().GetIndex() > m.GetEnd().GetIndex() {
				for index, placement := range indexToRecord {
					if index >= m.GetEnd().GetIndex() && index < m.GetStart().GetIndex() {
						placement.Index++
						nIndex[index+1] = placement
					}
				}
				nIndex[m.GetEnd().GetIndex()] = found
				found.Index = m.GetEnd().GetIndex()
				indexToRecord = nIndex
			}
		}
	}

	nPlacements := make([]*pb.Placement, len(indexToRecord))
	for i := int32(1); i <= int32(len(indexToRecord)); i++ {
		nPlacements[i-1] = indexToRecord[i]
	}
	return &pb.OrganisationSnapshot{Placements: nPlacements}
}

func TestSnapshotDiff(t *testing.T) {
	type test struct {
		start *pb.OrganisationSnapshot
		end   *pb.OrganisationSnapshot
	}

	tests := []test{
		{
			start: &pb.OrganisationSnapshot{
				Placements: []*pb.Placement{
					{
						Iid:   1234,
						Index: 1,
						Space: "Shelves",
						Unit:  1,
					},
					{
						Iid:   1235,
						Index: 2,
						Space: "Shelves",
						Unit:  1,
					},
				},
			},
			end: &pb.OrganisationSnapshot{
				Placements: []*pb.Placement{
					{
						Iid:   1234,
						Index: 2,
						Space: "Shelves",
						Unit:  1,
					},
					{
						Iid:   1235,
						Index: 1,
						Space: "Shelves",
						Unit:  1,
					},
				},
			},
		},
	}

	for _, test := range tests {
		diff := getSnapshotDiff(test.start, test.end)

		nsnap := applyMoves(test.start, diff)
		if getHash(test.end.GetPlacements()) != getHash(nsnap.GetPlacements()) {
			t.Errorf("Moves failed: \nstart    %v\nexpected %v\ngot      %v\nmoves: %v\n %v vs %v", test.start, test.end, nsnap, diff, getHash(test.end.GetPlacements()), getHash(nsnap.GetPlacements()))
		}
	}
}

func TestSetSnapshotName(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 12, Labels: []*pbd.Label{{Name: "BBB"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name: "testing",
						Foldersets: []*pb.FolderSet{
							{
								Name:   "testing",
								Folder: 12,
								Index:  1,
								Sort:   pb.Sort_LABEL_CATNO,
							}},
						Spaces: []*pb.Space{
							{
								Name:  "Main Shelves",
								Units: 1,
								Width: 100,
							}},
					},
				},
			},
		},
	})

	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}

	_, err = s.SetOrgSnapshot(ctx, &pb.SetOrgSnapshotRequest{OrgName: "testing", Name: "atestname", Date: org.GetSnapshot().GetDate()})
	if err != nil {
		t.Errorf("Unable to set org snapshot: %v", err)
	}

	org2, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing", Name: "atestname"})
	if err != nil {
		t.Errorf("Unable to get org from name: %v", err)
	}

	if getHash(org.GetSnapshot().GetPlacements()) != getHash(org2.GetSnapshot().GetPlacements()) {
		t.Errorf("Expected was not received\n%v\n%v", org2, org)
	}

}
