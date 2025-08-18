package server

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestConfigUpdate_UpdatesTime(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	c1, err := s.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		t.Fatalf("Unable to get state: %v", err)
	}

	nconfig := &pb.GramophileConfig{Basis: pb.Basis_GRAMOPHILE}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("Bad initial config set: %v", err)
	}

	c2, err := s.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		t.Fatalf("Unable to get state: %v", err)
	}

	if c1.GetLastConfigUpdate() == c2.GetLastConfigUpdate() {
		t.Errorf("Collection sync time was not updated: %v (%v)", c2.GetLastConfigUpdate(), time.Unix(0, c2.GetLastConfigUpdate()))
	}
}

func TestConfigUpdate_FailsOnMissingField(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	nconfig := &pb.GramophileConfig{
		Basis:            pb.Basis_GRAMOPHILE,
		GoalFolderConfig: &pb.GoalFolderConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err == nil {
		t.Errorf("Set config should have failed on missing field")
	}
}

func TestConfigUpdate_FailsOnBadUser(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Name: "Goal Folder", Id: 12}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("can't init save user: %v", err)
	}
	s := Server{d: d, di: di, qc: qc}

	ctx = getTestContext(1234)

	nconfig := &pb.GramophileConfig{
		Basis: pb.Basis_GRAMOPHILE, GoalFolderConfig: &pb.GoalFolderConfig{Enabled: pb.Enabled_ENABLED_ENABLED}}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err == nil {
		t.Errorf("Set config should have failed on missing field")
	}
}

func TestConfigUpdate_CreateWantlists(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	nconfig := &pb.GramophileConfig{
		Basis:       pb.Basis_GRAMOPHILE,
		WantsConfig: &pb.WantsConfig{MintUpWantList: true, DigitalWantList: true}}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("Bad initial config set: %v", err)
	}

	qc.FlushQueue(ctx)

	_, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{
		Name: "digital_wantlist",
	})
	if err != nil {
		t.Errorf("Unable to get digital wnatlist")
	}

	_, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{
		Name: "mint_up_wantlist",
	})
	if err != nil {
		t.Errorf("Unable to get mint_up wnatlist")
	}

}

func TestConfigUpdate_CreateFolders(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	nconfig := &pb.GramophileConfig{
		Basis:         pb.Basis_GRAMOPHILE,
		CreateFolders: pb.Create_AUTOMATIC,
		Validations:   []*pb.ValidationRule{{ValidationStrategy: pb.ValidationStrategy_LISTEN_TO_VALIDATE}},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("Bad initial config set: %v", err)
	}

	qc.FlushQueue(ctx)

	folders, err := di.GetUserFolders(ctx)
	if err != nil {
		t.Fatalf("Bad get folders: %v", err)
	}

	if len(folders) == 0 {
		t.Errorf("No folders created")
	}

}
