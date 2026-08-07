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

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

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

func TestConfigIncludesFloats(t *testing.T) {
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
		WantsConfig: &pb.WantsConfig{
			Origin:          pb.WantsBasis_WANTS_GRAMOPHILE,
			Existing:        pb.WantsExisting_EXISTING_LIST,
			DigitalWantList: true,
		},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("Bad initial config set: %v", err)
	}

	config, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Bad get user: %v", err)
	}

	wantlists := config.GetUser().GetConfig().GetWantsListConfig().GetWantlists()

	if len(wantlists) != 2 {
		t.Errorf("We are not returning the created wantlists: %v", wantlists)
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

func TestConfigUpdate_OrganisationValidationErrors(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	// 1. Missing width mandate error propagation (FailedPrecondition)
	nconfig := &pb.GramophileConfig{
		Basis: pb.Basis_GRAMOPHILE,
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:    "Org1",
					Density: pb.Density_WIDTH,
				},
			},
		},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err == nil {
		t.Fatalf("Expected error for missing width mandate, got nil")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Expected status code FailedPrecondition (%v), got: %v", codes.FailedPrecondition, status.Code(err))
	}

	// 2. Duplicate organisation name error propagation (AlreadyExists)
	nconfig2 := &pb.GramophileConfig{
		Basis: pb.Basis_GRAMOPHILE,
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{Name: "Org1"},
				{Name: "Org1"},
			},
		},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig2})
	if err == nil {
		t.Fatalf("Expected error for duplicate org name, got nil")
	}
	if status.Code(err) != codes.AlreadyExists {
		t.Errorf("Expected status code AlreadyExists (%v), got: %v", codes.AlreadyExists, status.Code(err))
	}

	// 3. Duplicate folder mapping error propagation (FailedPrecondition)
	nconfig3 := &pb.GramophileConfig{
		Basis: pb.Basis_GRAMOPHILE,
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name: "Org1",
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
					},
				},
				{
					Name: "Org2",
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
					},
				},
			},
		},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig3})
	if err == nil {
		t.Fatalf("Expected error for duplicate folder mapping, got nil")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Errorf("Expected status code FailedPrecondition (%v), got: %v", codes.FailedPrecondition, status.Code(err))
	}
}

func TestConfigUpdate_TriggersQueueOnValidConfig(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}

	s := Server{d: d, di: di, qc: qc}

	nconfig := &pb.GramophileConfig{
		Basis: pb.Basis_GRAMOPHILE,
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name: "ValidOrg",
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
					},
				},
			},
		},
	}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: nconfig})
	if err != nil {
		t.Fatalf("SetConfig failed on valid organisation config: %v", err)
	}

	elems, err := qc.List(ctx, &pb.ListRequest{})
	if err != nil {
		t.Fatalf("Unable to list queue elements: %v", err)
	}

	if len(elems.GetElements()) == 0 {
		t.Errorf("Expected task queue elements to be enqueued on valid config submission, got 0 elements")
	}
}

