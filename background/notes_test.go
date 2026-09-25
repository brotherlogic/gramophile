package background

import (
	"context"
	"fmt"
	"log"
	"sort"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/config"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/org"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	protov2 "google.golang.org/protobuf/proto"
)

func TestMovePrint(t *testing.T) {
	ctx := getTestContext(123)

	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{
		PrintMoveConfig: &pb.PrintMoveConfig{
			Context: 1,
			Enabled: pb.Enabled_ENABLED_ENABLED,
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

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "a", Artists: []*pbd.Artist{{Name: "arta"}}, InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Name: "aaa"}}}}, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, mr, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "c", Artists: []*pbd.Artist{{Name: "artc"}}, InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Name: "ccc"}}}}, &db.SaveOptions{})

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "d", Artists: []*pbd.Artist{{Name: "artd"}}, InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Name: "aaa"}}}}, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "e", Artists: []*pbd.Artist{{Name: "arte"}}, InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Name: "ccc"}}}}, &db.SaveOptions{})

	org1, err := org.GetOrgSwallow(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "First",
		Foldersets: []*pb.FolderSet{{Folder: 1}},
	}, su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", org1)

	org2, err := org.GetOrgSwallow(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "Second",
		Foldersets: []*pb.FolderSet{{Folder: 2}},
	}, su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "Second", org2)
	log.Printf("Saved snapshot: %v", org2)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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
	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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
			Enabled: pb.Enabled_ENABLED_ENABLED,
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

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "a", Artists: []*pbd.Artist{{Name: "arta"}}, InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Name: "aaa"}}}}, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, mr, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "c", Artists: []*pbd.Artist{{Name: "artc"}}, InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Name: "ccc"}}}}, &db.SaveOptions{})

	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "d", Artists: []*pbd.Artist{{Name: "artd"}}, InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Name: "aaa"}}}}, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Title: "e", Artists: []*pbd.Artist{{Name: "arte"}}, InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Name: "ccc"}}}}, &db.SaveOptions{})

	org1, err := org.GetOrgSwallow(b.db).BuildSnapshot(ctx, su, &pb.Organisation{
		Name:       "First",
		Foldersets: []*pb.FolderSet{{Folder: 1}},
	}, su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("Bad org build: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", org1)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), mr, &pb.Intent{NewFolder: 1}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
	if err == nil {
		t.Fatalf("Intent was processed: %v", err)
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
	b.db.SaveWantlist(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, &pb.Wantlist{Name: "mint_up_wantlist"})

	err := b.ProcessKeep(ctx, di, &pb.Record{}, &pb.Intent{
		Keep:    pb.KeepStatus_MINT_UP_KEEP,
		MintIds: []int64{124},
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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
	}, su, []*pbd.Field{{Id: 10, Name: "Keep"}}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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

	err := b.ProcessKeep(ctx, di, &pb.Record{}, &pb.Intent{Keep: pb.KeepStatus_MINT_UP_KEEP}, su, []*pbd.Field{}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Should have failed with :Failed Precondition %v", err)
	}
}

func TestMintUpKeep_Reset(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{}}

	err := b.ProcessKeep(ctx, di, &pb.Record{Release: &pbd.Release{InstanceId: 12345}, KeepStatus: pb.KeepStatus_DIGITAL_KEEP}, &pb.Intent{Keep: pb.KeepStatus_RESET}, su, []*pbd.Field{{Id: 10, Name: "Keep"}}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
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

	b.db.SaveWantlist(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, &pb.Wantlist{Name: "testing", Entries: []*pb.WantlistEntry{{Id: 123}}})

	err := b.ProcessScore(ctx, di, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234}}, &pb.Intent{NewScore: 3}, su, []*pbd.Field{})
	if err != nil {
		t.Fatalf("Unable to process score: %v", err)
	}
}

func TestScoreRecord_ValidateMapping(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	su := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}, Config: &pb.GramophileConfig{
		ScoreConfig: &pb.ScoreConfig{
			TopRange:    100,
			BottomRange: 1,
		},
	}}

	err := b.ProcessScore(ctx, di, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234}}, &pb.Intent{NewScore: 67}, su, []*pbd.Field{})
	if err != nil {
		t.Fatalf("Unable to process score: %v", err)
	}

	r, err := b.db.GetRecord(ctx, 123, 1234)
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetRelease().GetRating() != 4 {
		t.Errorf("Rating was not set correctly: %v", r.GetRelease().GetRating())
	}
}

func TestPurchaseLocation(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Purchase Location"}}}
	su := &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Auth:   &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{}}
	b.db.SaveUser(ctx, su)
	b.db.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234}})

	err := b.ProcessIntents(ctx, di, &pb.Record{Release: &pbd.Release{InstanceId: 1234}}, &pb.Intent{
		PurchaseLocation: "bounce",
	}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		// Do nothing
		return nil, nil
	})
	if err != nil {
		t.Fatalf("Unable to process purchase location: %v", err)
	}

	r, err := b.db.GetRecord(ctx, 123, 1234)
	if err != nil {
		t.Fatalf("Unable to get record: %v", err)
	}

	if r.GetPurchaseLocation() != "bounce" {
		t.Errorf("Location was not set correctly: %v", r)
	}
}

func TestProcessSetFolder_ComprehensivePrinting_CrossOrg(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			PrintMoveConfig: &pb.PrintMoveConfig{
				Enabled: pb.Enabled_ENABLED_ENABLED,
				Context: 1,
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:    "First",
						Density: pb.Density_COUNT,
						Spaces: []*pb.Space{
							{Name: "Main", Units: 10, Width: 1},
						},
						Foldersets: []*pb.FolderSet{{Folder: 1, Sort: pb.Sort_LABEL_CATNO}},
					},
					{
						Name:    "Second",
						Density: pb.Density_COUNT,
						Spaces: []*pb.Space{
							{Name: "Main", Units: 10, Width: 1},
						},
						Foldersets: []*pb.FolderSet{{Folder: 2, Sort: pb.Sort_LABEL_CATNO}},
					},
				},
			},
		},
	}
	if err := b.db.SaveUser(ctx, su); err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	r1 := &pb.Record{Release: &pbd.Release{Title: "R1", InstanceId: 1, FolderId: 1, Labels: []*pbd.Label{{Catno: "100"}}}}
	r2 := &pb.Record{Release: &pbd.Release{Title: "R2", InstanceId: 2, FolderId: 1, Labels: []*pbd.Label{{Catno: "200"}}}}
	r3 := &pb.Record{Release: &pbd.Release{Title: "R3", InstanceId: 3, FolderId: 1, Labels: []*pbd.Label{{Catno: "300"}}}}
	r4 := &pb.Record{Release: &pbd.Release{Title: "R4", InstanceId: 4, FolderId: 2, Labels: []*pbd.Label{{Catno: "050"}}}}
	r5 := &pb.Record{Release: &pbd.Release{Title: "R5", InstanceId: 5, FolderId: 2, Labels: []*pbd.Label{{Catno: "250"}}}}

	b.db.SaveRecord(ctx, 123, r1, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r2, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r3, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r4, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r5, &db.SaveOptions{})

	orglogic, err := org.GetOrg(b.db)
	if err != nil {
		t.Fatalf("GetOrg error: %v", err)
	}
	snap1, err := orglogic.BuildSnapshot(ctx, su, su.GetConfig().GetOrganisationConfig().GetOrganisations()[0], su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("BuildSnapshot 1 error: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "First", snap1)

	snap2, err := orglogic.BuildSnapshot(ctx, su, su.GetConfig().GetOrganisationConfig().GetOrganisations()[1], su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("BuildSnapshot 2 error: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "Second", snap2)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), r2, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents error: %v", err)
	}

	moves, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("LoadPrintMoves error: %v", err)
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].GetIndex() < moves[j].GetIndex()
	})

	if len(moves) != 3 {
		t.Fatalf("Expected 3 print moves, got %d: %+v", len(moves), moves)
	}

	if moves[0].GetType() != pb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE || moves[0].GetIid() != 3 {
		t.Errorf("Move 0 should be Exit shuffle of Rec 3, got %+v", moves[0])
	}
	if moves[1].GetType() != pb.PrintMoveType_PRINT_MOVE_TYPE_MOVE || moves[1].GetIid() != 2 {
		t.Errorf("Move 1 should be Main move of Rec 2, got %+v", moves[1])
	}
	if moves[2].GetType() != pb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE || moves[2].GetIid() != 5 {
		t.Errorf("Move 2 should be Dest shuffle of Rec 5, got %+v", moves[2])
	}

	if moves[0].GetIndex() >= moves[1].GetIndex() || moves[1].GetIndex() >= moves[2].GetIndex() {
		t.Errorf("Expected strictly monotonic increasing indexes, got: %v, %v, %v",
			moves[0].GetIndex(), moves[1].GetIndex(), moves[2].GetIndex())
	}
}

func TestProcessSetFolder_SameOrg(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			PrintMoveConfig: &pb.PrintMoveConfig{
				Enabled: pb.Enabled_ENABLED_ENABLED,
				Context: 1,
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:    "SingleOrg",
						Density: pb.Density_COUNT,
						Spaces: []*pb.Space{
							{Name: "Main", Units: 10, Width: 1},
						},
						Foldersets: []*pb.FolderSet{
							{Folder: 1, Index: 1, Sort: pb.Sort_LABEL_CATNO},
							{Folder: 2, Index: 2, Sort: pb.Sort_LABEL_CATNO},
						},
					},
				},
			},
		},
	}
	if err := b.db.SaveUser(ctx, su); err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	r1 := &pb.Record{Release: &pbd.Release{Title: "R1", InstanceId: 10, FolderId: 1, Labels: []*pbd.Label{{Catno: "100"}}}}
	r2 := &pb.Record{Release: &pbd.Release{Title: "R2", InstanceId: 20, FolderId: 1, Labels: []*pbd.Label{{Catno: "200"}}}}
	r3 := &pb.Record{Release: &pbd.Release{Title: "R3", InstanceId: 30, FolderId: 2, Labels: []*pbd.Label{{Catno: "300"}}}}

	b.db.SaveRecord(ctx, 123, r1, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r2, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, r3, &db.SaveOptions{})

	orglogic, err := org.GetOrg(b.db)
	if err != nil {
		t.Fatalf("GetOrg error: %v", err)
	}
	snap, err := orglogic.BuildSnapshot(ctx, su, su.GetConfig().GetOrganisationConfig().GetOrganisations()[0], su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("BuildSnapshot error: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "SingleOrg", snap)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), r1, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents error: %v", err)
	}

	moves, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("LoadPrintMoves error: %v", err)
	}

	var mainMoves []*pb.PrintMove
	var shuffleMoves []*pb.PrintMove
	seenIids := make(map[int64]bool)

	for _, m := range moves {
		if seenIids[m.GetIid()] {
			t.Errorf("Duplicate move generated for iid %v", m.GetIid())
		}
		seenIids[m.GetIid()] = true

		if m.GetType() == pb.PrintMoveType_PRINT_MOVE_TYPE_MOVE {
			mainMoves = append(mainMoves, m)
		} else if m.GetType() == pb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE {
			shuffleMoves = append(shuffleMoves, m)
		}
	}

	if len(mainMoves) != 1 {
		t.Fatalf("Expected exactly 1 main move, got %d: %+v", len(mainMoves), mainMoves)
	}
	if mainMoves[0].GetIid() != 10 {
		t.Errorf("Main move should be for r1 (iid 10), got %v", mainMoves[0].GetIid())
	}
	if len(shuffleMoves) != 1 {
		t.Fatalf("Expected exactly 1 shuffle move, got %d: %+v", len(shuffleMoves), shuffleMoves)
	}
	if shuffleMoves[0].GetIid() != 20 {
		t.Errorf("Shuffle move should be for r2 (iid 20), got %v", shuffleMoves[0].GetIid())
	}
}

func TestProcessSetFolder_FromNew(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			PrintMoveConfig: &pb.PrintMoveConfig{
				Enabled: pb.Enabled_ENABLED_ENABLED,
				Context: 1,
			},
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:    "Destination",
						Density: pb.Density_COUNT,
						Spaces: []*pb.Space{
							{Name: "Main", Units: 10, Width: 1},
						},
						Foldersets: []*pb.FolderSet{{Folder: 2, Sort: pb.Sort_LABEL_CATNO}},
					},
				},
			},
		},
	}
	if err := b.db.SaveUser(ctx, su); err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	rExisting := &pb.Record{Release: &pbd.Release{Title: "Existing", InstanceId: 100, FolderId: 2, Labels: []*pbd.Label{{Catno: "200"}}}}
	rNew := &pb.Record{Release: &pbd.Release{Title: "NewRec", InstanceId: 200, FolderId: 0, Labels: []*pbd.Label{{Catno: "100"}}}}

	b.db.SaveRecord(ctx, 123, rExisting, &db.SaveOptions{})
	b.db.SaveRecord(ctx, 123, rNew, &db.SaveOptions{})

	orglogic, err := org.GetOrg(b.db)
	if err != nil {
		t.Fatalf("GetOrg error: %v", err)
	}
	snap, err := orglogic.BuildSnapshot(ctx, su, su.GetConfig().GetOrganisationConfig().GetOrganisations()[0], su.GetConfig().GetOrganisationConfig())
	if err != nil {
		t.Fatalf("BuildSnapshot error: %v", err)
	}
	b.db.SaveSnapshot(ctx, su, "Destination", snap)

	err = b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), rNew, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents error: %v", err)
	}

	moves, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("LoadPrintMoves error: %v", err)
	}

	sort.Slice(moves, func(i, j int) bool {
		return moves[i].GetIndex() < moves[j].GetIndex()
	})

	for _, m := range moves {
		if m.GetType() == pb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE && m.GetOrigin().GetLocationName() == "New" {
			t.Errorf("Found invalid exit shuffle from New: %+v", m)
		}
	}

	if len(moves) != 2 {
		t.Fatalf("Expected 2 moves (1 main move, 1 dest shuffle), got %d: %+v", len(moves), moves)
	}
	if moves[0].GetType() != pb.PrintMoveType_PRINT_MOVE_TYPE_MOVE || moves[0].GetIid() != 200 {
		t.Errorf("First move should be Main move of rNew (200), got %+v", moves[0])
	}
	if moves[1].GetType() != pb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE || moves[1].GetIid() != 100 {
		t.Errorf("Second move should be Dest shuffle of rExisting (100), got %+v", moves[1])
	}
}

func TestProcessSetFolder_PrintMoveDisabled(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			PrintMoveConfig: &pb.PrintMoveConfig{
				Enabled: pb.Enabled_ENABLED_DISABLED,
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
		},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{Title: "DisabledTest", InstanceId: 777, FolderId: 1}}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	err := b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), r, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents failed: %v", err)
	}

	rec, err := b.db.GetRecord(ctx, 123, 777)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if rec.GetRelease().GetFolderId() != 2 {
		t.Errorf("Record folder should have been updated to 2, got %v", rec.GetRelease().GetFolderId())
	}

	moves, err := b.db.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("LoadPrintMoves failed: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("Expected 0 print moves when disabled, got %d: %+v", len(moves), moves)
	}
}

type failingSnapshotDB struct {
	db.Database
}

func (f *failingSnapshotDB) GetLatestSnapshot(ctx context.Context, userid int32, org string) (*pb.OrganisationSnapshot, error) {
	return nil, fmt.Errorf("forced snapshot failure")
}

func (f *failingSnapshotDB) GetRecords(ctx context.Context, userid int32) ([]int64, error) {
	return nil, fmt.Errorf("forced get records failure")
}

func TestProcessSetFolder_SnapshotFailure(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()

	su := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{
			PrintMoveConfig: &pb.PrintMoveConfig{
				Enabled: pb.Enabled_ENABLED_ENABLED,
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
		},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{Title: "FailureTest", InstanceId: 888, FolderId: 1}}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	origDB := b.db
	b.db = &failingSnapshotDB{Database: origDB}

	err := b.ProcessIntents(ctx, discogs.GetTestClient().ForUser(&pbd.User{DiscogsUserId: 123}), r, &pb.Intent{NewFolder: 2}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents should halt move enqueueing cleanly on snapshot failure, but returned error: %v", err)
	}

	moves, err := origDB.LoadPrintMoves(ctx, 123)
	if err != nil {
		t.Fatalf("LoadPrintMoves failed: %v", err)
	}
	if len(moves) != 0 {
		t.Errorf("Expected 0 moves when snapshot fails, got %d", len(moves))
	}
}

type testFieldTracker struct {
	*discogs.TestDiscogsClient
	calls []testFieldCall
}

type testFieldCall struct {
	release *pbd.Release
	field   int
	val     string
}

func (t *testFieldTracker) SetField(ctx context.Context, r *pbd.Release, fnum int, value string) error {
	t.calls = append(t.calls, testFieldCall{release: r, field: fnum, val: value})
	return t.TestDiscogsClient.SetField(ctx, r, fnum, value)
}

func TestProcessSetPackageScore_MissingField(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123}
	su := &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Auth:   &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{InstanceId: 1001, Id: 2001}}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	// Fields missing the "Package" field
	fields := []*pbd.Field{{Id: 1, Name: "OtherField"}}
	err := b.ProcessSetPackageScore(ctx, di, r, &pb.Intent{PackageScore: protov2.Int32(4)}, su, fields)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("Expected codes.FailedPrecondition, got %v (code: %v)", err, status.Code(err))
	}
}

func TestProcessSetPackageScore_Success(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	tracker := &testFieldTracker{TestDiscogsClient: &discogs.TestDiscogsClient{UserId: 123}}
	su := &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Auth:   &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{InstanceId: 1001, Id: 2001}}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	fields := []*pbd.Field{{Id: 42, Name: config.PACKAGE_FIELD}}
	err := b.ProcessSetPackageScore(ctx, tracker, r, &pb.Intent{PackageScore: protov2.Int32(4)}, su, fields)
	if err != nil {
		t.Fatalf("ProcessSetPackageScore failed: %v", err)
	}

	if len(tracker.calls) != 1 {
		t.Fatalf("Expected 1 SetField call, got %d", len(tracker.calls))
	}
	if tracker.calls[0].field != 42 || tracker.calls[0].val != "4" {
		t.Errorf("Unexpected SetField call: %+v", tracker.calls[0])
	}

	saved, err := b.db.GetRecord(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if saved.GetPackageScore() != 4 {
		t.Errorf("Expected package_score 4, got %v", saved.GetPackageScore())
	}

	// Also verify wiring through ProcessIntents with Discogs fields
	diWithFields := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 42, Name: config.PACKAGE_FIELD}},
	}
	rWiring := &pb.Record{Release: &pbd.Release{InstanceId: 1002, Id: 2002}}
	b.db.SaveRecord(ctx, 123, rWiring, &db.SaveOptions{})
	err = b.ProcessIntents(ctx, diWithFields, rWiring, &pb.Intent{PackageScore: protov2.Int32(5)}, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return nil, nil
	})
	if err != nil {
		t.Fatalf("ProcessIntents failed: %v", err)
	}
	savedWiring, err := b.db.GetRecord(ctx, 123, 1002)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if savedWiring.GetPackageScore() != 5 {
		t.Errorf("Expected package_score 5 via ProcessIntents, got %v", savedWiring.GetPackageScore())
	}
}

func TestProcessSetPackageScore_Reset(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	tracker := &testFieldTracker{TestDiscogsClient: &discogs.TestDiscogsClient{UserId: 123}}
	su := &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Auth:   &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{InstanceId: 1001, Id: 2001}, PackageScore: 3}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	fields := []*pbd.Field{{Id: 42, Name: config.PACKAGE_FIELD}}
	err := b.ProcessSetPackageScore(ctx, tracker, r, &pb.Intent{PackageScore: protov2.Int32(-1)}, su, fields)
	if err != nil {
		t.Fatalf("ProcessSetPackageScore reset failed: %v", err)
	}

	if len(tracker.calls) != 1 {
		t.Fatalf("Expected 1 SetField call, got %d", len(tracker.calls))
	}
	if tracker.calls[0].field != 42 || tracker.calls[0].val != "" {
		t.Errorf("Unexpected SetField call on reset: %+v", tracker.calls[0])
	}

	saved, err := b.db.GetRecord(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if saved.GetPackageScore() != 0 {
		t.Errorf("Expected package_score 0 after reset, got %v", saved.GetPackageScore())
	}
}

func TestProcessSetPackageScore_Untouched(t *testing.T) {
	ctx := getTestContext(123)
	b := GetTestBackgroundRunner()
	tracker := &testFieldTracker{TestDiscogsClient: &discogs.TestDiscogsClient{UserId: 123}}
	su := &pb.StoredUser{
		User:   &pbd.User{DiscogsUserId: 123},
		Auth:   &pb.GramophileAuth{Token: "123"},
		Config: &pb.GramophileConfig{},
	}
	b.db.SaveUser(ctx, su)

	r := &pb.Record{Release: &pbd.Release{InstanceId: 1001, Id: 2001}, PackageScore: 3}
	b.db.SaveRecord(ctx, 123, r, &db.SaveOptions{})

	fields := []*pbd.Field{{Id: 42, Name: config.PACKAGE_FIELD}}

	// 1. Nil PackageScore intent should be a no-op
	err := b.ProcessSetPackageScore(ctx, tracker, r, &pb.Intent{}, su, fields)
	if err != nil {
		t.Fatalf("ProcessSetPackageScore failed on nil intent: %v", err)
	}
	if len(tracker.calls) != 0 {
		t.Errorf("Expected 0 SetField calls for nil intent, got %d", len(tracker.calls))
	}
	saved, err := b.db.GetRecord(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if saved.GetPackageScore() != 3 {
		t.Errorf("Expected package_score to remain 3 on nil intent, got %v", saved.GetPackageScore())
	}

	// 2. Out-of-bounds score (< -1 or > 5) should be a no-op
	err = b.ProcessSetPackageScore(ctx, tracker, r, &pb.Intent{PackageScore: protov2.Int32(6)}, su, fields)
	if err != nil {
		t.Fatalf("ProcessSetPackageScore failed on out-of-range: %v", err)
	}
	err = b.ProcessSetPackageScore(ctx, tracker, r, &pb.Intent{PackageScore: protov2.Int32(-2)}, su, fields)
	if err != nil {
		t.Fatalf("ProcessSetPackageScore failed on out-of-range: %v", err)
	}
	if len(tracker.calls) != 0 {
		t.Errorf("Expected 0 SetField calls for out-of-range intents, got %d", len(tracker.calls))
	}
	saved, err = b.db.GetRecord(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}
	if saved.GetPackageScore() != 3 {
		t.Errorf("Expected package_score to remain 3 on out-of-range, got %v", saved.GetPackageScore())
	}
}

