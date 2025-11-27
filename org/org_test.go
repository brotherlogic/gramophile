package org

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	"google.golang.org/grpc/metadata"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func TestJoinOrdering(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC", Id: 1}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 13, Labels: []*pbd.Label{{Name: "BBB", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1236, FolderId: 14, Labels: []*pbd.Label{{Name: "AAA", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
		Name: "testing",
		Foldersets: []*pb.FolderSet{
			{
				Name:   "testing",
				Folder: 12,
				Index:  1,
				Sort:   pb.Sort_LABEL_CATNO,
			},
			{
				Name:   "testing2",
				Folder: 14,
				Index:  1,
				Sort:   pb.Sort_LABEL_CATNO,
			},
			{
				Name:   "testing3",
				Folder: 13,
				Index:  2,
				Sort:   pb.Sort_LABEL_CATNO,
			}},
		Spaces: []*pb.Space{
			{
				Name:  "Main Shelves",
				Units: 1,
				Width: 100,
			}},
	}, &pb.OrganisationConfig{})

	for _, entry := range snap.GetPlacements() {
		if entry.GetIndex() == 1 {
			if entry.GetIid() != 1236 {
				t.Errorf("AAA was not first: %v", snap.GetPlacements())
			}
		}

		if entry.GetIndex() == 2 {
			if entry.GetIid() != 1234 {
				t.Errorf("CCC was not second: %v", snap.GetPlacements())
			}
		}

		if entry.GetIndex() == 3 {
			if entry.GetIid() != 1235 {
				t.Errorf("BBB was not third: %v", snap.GetPlacements())
			}
		}
	}

}

func TestLabelRanking(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA", Id: 1}, {Name: "ZZZ", Id: 2}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
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
	}, &pb.OrganisationConfig{
		LabelRanking: []*pb.LabelWeight{{LabelId: 2, Weight: 2.0}},
	})

	// First record should come after the second
	if len(snap.GetPlacements()) != 2 {
		t.Fatalf("SHould be two placements: %v", snap)
	}

	if snap.GetPlacements()[0].GetIid() == 1234 {
		t.Errorf("1234 should come after 1235: %v", snap)
	}
}

func TestOrderByArtist(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 1234,
			FolderId:   12,
			Artists:    []*pbd.Artist{{Name: "The Beatles"}},
			Labels:     []*pbd.Label{{Name: "AAA", Id: 1}, {Name: "ZZZ", Id: 2}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 1235,
			FolderId:   12,
			Artists:    []*pbd.Artist{{Name: "The Rolling Stones"}},
			Labels:     []*pbd.Label{{Name: "CCC", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId: 1236,
			FolderId:   12,
			Artists:    []*pbd.Artist{{Name: "Black Sabbath"}},
			Labels:     []*pbd.Label{{Name: "CCC", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic, _ := GetOrg(d)
	snap, err := orglogic.BuildSnapshot(ctx, user, &pb.Organisation{
		Name: "testing",
		Foldersets: []*pb.FolderSet{
			{
				Name:   "testing",
				Folder: 12,
				Index:  1,
				Sort:   pb.Sort_ARTIST_YEAR,
			}},
		Spaces: []*pb.Space{
			{
				Name:  "Main Shelves",
				Units: 1,
				Width: 100,
			}},
	}, &pb.OrganisationConfig{
		LabelRanking: []*pb.LabelWeight{{LabelId: 2, Weight: 2.0}},
	})

	// First record should come after the second
	if len(snap.GetPlacements()) != 3 {
		t.Fatalf("SHould be two placements: %v", snap)
	}

	if snap.GetPlacements()[0].GetIid() != 1234 {
		t.Errorf("1234 should be first: %v", snap.GetPlacements()[0])
	}

	if snap.GetPlacements()[1].GetIid() != 1236 {
		t.Errorf("1236 should be second: %v", snap.GetPlacements()[1])
	}

	if snap.GetPlacements()[2].GetIid() != 1235 {
		t.Errorf("1235 should be third: %v", snap.GetPlacements()[2])
	}
}
