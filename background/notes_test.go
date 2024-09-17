package background

import (
	"context"
	"log"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	}, su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", org1)

	org2, err := org.GetOrg(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "Second",
		Foldersets: []*pb.FolderSet{{Folder: 2}},
	}, su.GetConfig().GetOrganisationConfig())
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

	if move.GetOrigin().GetAfter()[0].GetRecord() != "artc - c" {
		t.Errorf("Bad after: %v", move.GetOrigin().GetAfter())
	}

	if len(move.GetDestination().GetBefore()) == 0 {
		t.Fatalf("Missing destination: %v", move.GetDestination())
	}
	if move.GetDestination().GetBefore()[0].GetRecord() != "artd - d" {
		t.Errorf("Bad dest before: %v", move.GetDestination().GetBefore())
	}
	if move.GetDestination().GetAfter()[0].GetRecord() != "arte - e" {
		t.Errorf("Bad dest before: %v", move.GetDestination().GetAfter())
	}

	// Also test that if we re-move it we get a nil return
	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123")
	if err != nil {
		t.Fatalf("Bad intent processing: %v", err)
	}
}

func TestMovePrint_MissingOrgorigin(t *testing.T) {
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
			},
		},
	}}
	err := b.db.SaveUser(context.Background(), su)
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	mr := &pb.Record{Release: &pbd.Release{Title: "b", Artists: []*pbd.Artist{{Name: "artb"}}, InstanceId: 2, FolderId: 2, Labels: []*pbd.Label{{Name: "bbb"}}}}

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "a", Artists: []*pbd.Artist{{Name: "arta"}}, InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, mr)
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "c", Artists: []*pbd.Artist{{Name: "artc"}}, InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Name: "ccc"}}}})

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "d", Artists: []*pbd.Artist{{Name: "artd"}}, InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Name: "aaa"}}}})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "e", Artists: []*pbd.Artist{{Name: "arte"}}, InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Name: "ccc"}}}})

	org1, err := org.GetOrg(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "First",
		Foldersets: []*pb.FolderSet{{Folder: 1}},
	}, su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", org1)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 1}, "123")
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

	if move.GetOrigin().GetAfter()[0].GetRecord() != "artc - c" {
		t.Errorf("Bad after: %v", move.GetOrigin().GetAfter())
	}

	if move.GetDestination().GetLocationName() != "Folder 2" {
		t.Errorf("Destination was not corrected: %v", move.GetDestination())
	}

	// Also test that if we re-move it we get a nil return
	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123")
	if err != nil {
		t.Fatalf("Bad intent processing: %v", err)
	}
}

func TestMintUpKeep_Success(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{
			MintUpWantList: true,
		},
	}}
	b.db.SaveWantlist(ctx, 123, &pb.Wantlist{Name: "mint_up_wantlist"})

	err := b.ProcessKeep(ctx, di, &pb.Record{}, &pb.Intent{
		Keep:    pb.KeepStatus_MINT_UP_KEEP,
		MintIds: []int64{124},
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}})
	if err != nil {
		t.Errorf("Unable to process keep: %v", err)
	}

	wl, err := b.db.LoadWantlist(ctx, 123, "mint_up_wantlist")
	if err != nil {
		t.Errorf("Unable to load wnatlist: %v", err)
	}
	if len(wl.GetEntries()) != 1 {
		t.Errorf("Want was not added: %v", wl)
	}

	// Add the same want
	err = b.ProcessKeep(ctx, di, &pb.Record{}, &pb.Intent{
		Keep:    pb.KeepStatus_MINT_UP_KEEP,
		MintIds: []int64{124},
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}})
	if err != nil {
		t.Errorf("Unable to process keep: %v", err)
	}

	wl, err = b.db.LoadWantlist(ctx, 123, "mint_up_wantlist")
	if err != nil {
		t.Errorf("Unable to load wnatlist: %v", err)
	}
	if len(wl.GetEntries()) != 1 {
		t.Errorf("Want was not added: %v", wl)
	}

	// Prepend an existing want
	err = b.ProcessKeep(ctx, di, &pb.Record{MintVersions: []int64{125}}, &pb.Intent{
		Keep:    pb.KeepStatus_MINT_UP_KEEP,
		MintIds: []int64{125},
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}})
	if err != nil {
		t.Errorf("Unable to process keep: %v", err)
	}

	wl, err = b.db.LoadWantlist(ctx, 123, "mint_up_wantlist")
	if err != nil {
		t.Errorf("Unable to load wnatlist: %v", err)
	}
	if len(wl.GetEntries()) != 2 {
		t.Errorf("Want was not added: %v", wl)
	}
}

func TestMintUpKeep_NoField(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{}}

	err := b.ProcessKeep(ctx, di, &pb.Record{}, &pb.Intent{Keep: pb.KeepStatus_MINT_UP_KEEP}, su, []*pbd.Field{})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Should have failed with :Failed Precondition %v", err)
	}
}

func TestMintUpKeep_Reset(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{}}

	err := b.ProcessKeep(ctx, di, &pb.Record{Release: &pbd.Release{InstanceId: 12345}, KeepStatus: pb.KeepStatus_DIGITAL_KEEP}, &pb.Intent{Keep: pb.KeepStatus_RESET}, su, []*pbd.Field{{Id: 10, Name: "Keep"}})
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}

	r, err := b.db.GetRecord(ctx, 123, 12345)
	if err != nil {
		t.Errorf("Bad records read: %v", err)
	}
	if r.GetKeepStatus() != pb.KeepStatus_KEEP_UNKNOWN {
		t.Errorf("Keep state was not updated")
	}
}

func TestScoreRecord_Wantlist(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{}}

	b.db.SaveWantlist(ctx, 123, &pb.Wantlist{Name: "testing", Entries: []*pb.WantlistEntry{{Id: 123}}})

	err := b.ProcessScore(ctx, di, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234}}, &pb.Intent{NewScore: 3}, su, []*pbd.Field{})
	if err != nil {
		t.Fatalf("Unable to process score: %v", err)
	}
}
