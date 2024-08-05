package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestMovePrint(t *testing.T) {
	ctx := getTestContext(123)

	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:       "First",
					Foldersets: []*pb.FolderSet{{Folder: 1}},
				},
				{
					Name:       "Second",
					Foldersets: []*pb.FolderSet{{Folder: 2}},
				},
			},
		},
	}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	mr := &pb.Record{Release: &pbd.Release{Title: "b", Artists: []*pbd.Artist{{Name: "artb"}}, InstanceId: 2, FolderId: 1, Labels: []*pbd.Label{{Name: "bbb"}}}}

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "a", Artists: []*pbd.Artist{{Name: "arta"}}, InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, mr)
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "c", Artists: []*pbd.Artist{{Name: "artc"}}, InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Name: "ccc"}}}})

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "d", Artists: []*pbd.Artist{{Name: "artd"}}, InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "e", Artists: []*pbd.Artist{{Name: "arte"}}, InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Name: "ccc"}}}})

	b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123")

	// That should have created one print entry
	v, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("Unable to get print moves: %v", err)
	}

	if len(v) != 1 {
		t.Fatalf("Wrong number of printed moves: %v", v)
	}
}
