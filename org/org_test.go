package org

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	"google.golang.org/grpc/metadata"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	rstore_client "github.com/brotherlogic/rstore/client"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func TestLabelRanking(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA", Id: 1}, {Name: "ZZZ", Id: 2}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}
	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1235, FolderId: 12, Labels: []*pbd.Label{{Name: "CCC", Id: 3}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}

	orglogic := GetOrg(d)
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
