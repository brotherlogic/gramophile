package background

import (
	"context"
	"log"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestMovePrint(t *testing.T) {
	ctx := getTestContext(123)

	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{
		PrintMoveConfig: &pb.PrintMoveConfig{
			Context: 1,
		},
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:       "First",
					Foldersets: []*pb.FolderSet{{Folder: 1, Sort: pb.Sort_LABEL_CATNO}},
				},
				{
					Name:       "Second",
					Foldersets: []*pb.FolderSet{{Folder: 2, Sort: pb.Sort_LABEL_CATNO}},
				},
			},
		},
	}}
	err := b.db.SaveUser(context.Background(), su)
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	mr := &pb.Record{Release: &pbd.Release{Title: "b", Artists: []*pbd.Artist{{Name: "artb"}}, InstanceId: 2, FolderId: 1, Labels: []*pbd.Label{{Name: "bbb"}}}}

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "a", Artists: []*pbd.Artist{{Name: "arta"}}, InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, mr)
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "c", Artists: []*pbd.Artist{{Name: "artc"}}, InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Name: "ccc"}}}})

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "d", Artists: []*pbd.Artist{{Name: "artd"}}, InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "e", Artists: []*pbd.Artist{{Name: "arte"}}, InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Name: "ccc"}}}})

	org1, err := org.GetOrg(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "First",
		Foldersets: []*pb.FolderSet{{Folder: 1}},
	})
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", org1)

	org2, err := org.GetOrg(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "Second",
		Foldersets: []*pb.FolderSet{{Folder: 2}},
	})
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "Second", org2)
	log.Printf("Saved snapshot: %v", org2)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123")
	if err != nil {
		t.Fatalf("Bad intent processing: %v", err)
	}

	// That should have created one print entry
	v, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("Unable to get print moves: %v", err)
	}

	if len(v) != 1 {
		t.Fatalf("Wrong number of printed moves: %v", v)
	}

	move := v[0]

	if move.GetOrigin().GetBefore()[0].GetRecord() != "arta - a" {
		t.Errorf("Bad before: %v", move.GetOrigin().GetBefore())
	}

	if len(move.GetDestination().GetBefore()) == 0 {
		t.Fatalf("Missing destination: %v", move.GetDestination())
	}
	if move.GetDestination().GetBefore()[0].GetRecord() != "artd - d" {
		t.Errorf("Bad after: %v", move.GetDestination().GetBefore())
	}
}
