package server

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/protobuf/proto"
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

func TestGetSnapshotHash(t *testing.T) {
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

	time.Sleep(time.Second * 2)

	org2, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "testing"})
	if err != nil {
		t.Fatalf("Unable to get second org version")
	}

	if org.GetSnapshot().GetDate() == org2.GetSnapshot().GetDate() || org.GetSnapshot().GetHash() != org2.GetSnapshot().GetHash() {
		t.Errorf("Hash or Date mismatch on second pull: %v vs %v", org.GetSnapshot(), org2.GetSnapshot())
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
