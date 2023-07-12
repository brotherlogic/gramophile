package server

import (
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestLabelOrdering(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB()
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

	s := Server{d: d, di: &discogs.TestDiscogsClient{}}

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
								Name:         "Main Shelves",
								Units:        1,
								RecordsWidth: 100,
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

	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && o.Iid != 1234 {
			t.Errorf("Bad placement: %v", org.GetSnapshot().GetPlacements())
		}
		if o.Index == 2 && o.Iid != 1236 {
			t.Errorf("Bad placement: %v", org.GetSnapshot().GetPlacements())
		}
		if o.Index == 3 && o.Iid != 1235 {
			t.Errorf("Bad Placment: %v", org.GetSnapshot().GetPlacements())
		}
	}
}

func TestLooseLayoutSupport(t *testing.T) {
	ctx := getTestContext(123)

	d := db.NewTestDB()
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveUser(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: &discogs.TestDiscogsClient{}}

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
								Name:         "Main Shelves",
								Units:        2,
								RecordsWidth: 100,
								Layout:       pb.Layout_LOOSE,
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

	for _, o := range org.GetSnapshot().GetPlacements() {
		if o.Index == 1 && (o.Iid != 1234 || o.Unit != 1) {
			t.Errorf("Bad placement: %v", org.GetSnapshot().GetPlacements())
		}
		if o.Index == 2 && (o.Iid != 1235 || o.Unit != 2) {
			t.Errorf("Bad Placment: %v", org.GetSnapshot().GetPlacements())
		}
	}
}
