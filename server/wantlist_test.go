package server

import (
	"log"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestGetWantsFromWantlist_hidden(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{
			Existing: pb.WantsExisting_EXISTING_LIST,
			Origin:   pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{}}}})
	if err != nil {
		t.Fatalf("Bad config set: %v", err)
	}

	config, err := s.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}

	config.GetUser().GetConfig().GetWantsListConfig().Wantlists = append(config.GetUser().GetConfig().GetWantsListConfig().Wantlists, &pb.StoredWantlist{
		Name:       "testing",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_INVISIBLE,
		Entries: []*pb.StoredWantlistEntry{
			{
				Id: 1234,
			},
		},
	})
	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{Config: config.GetUser().GetConfig()})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Flush out any queue stuff
	qc.FlushQueue(ctx)

	// Validate we have saved wantlists
	lists, err := s.ListWantlists(ctx, &pb.ListWantlistsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wantlsits: %v", err)
	}
	if len(lists.GetLists()) == 0 {
		t.Fatalf("No wantlists returned")
	}

	if lists.GetLists()[0].GetVisibility() != pb.WantlistVisibility_INVISIBLE {
		t.Fatalf("Wantlist visibility was not set: %v", lists.GetLists())
	}

	// We should be able to identify 1234 in wants
	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) == 0 {
		t.Fatalf("No wants listed")
	}

	if wants.GetWants()[0].GetWant().State != pb.WantState_HIDDEN {
		t.Errorf("Want was not hidden: %v -> %v", wants, wants.GetWants()[0].GetWant().GetState())
	}
}

func TestGetWantsFromWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{
			Existing: pb.WantsExisting_EXISTING_LIST,
			Origin:   pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{
			Wantlists: []*pb.StoredWantlist{
				{
					Name: "testing",
					Type: pb.WantlistType_ONE_BY_ONE,
					Entries: []*pb.StoredWantlistEntry{
						{
							Id: 1234,
						},
					},
				},
			},
		},
	}})

	// Flush out any queue stuff
	qc.FlushQueue(ctx)

	// We should be able to identify 1234 in wants
	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) == 0 {
		t.Errorf("No wants listed")
	}
}

func TestSaveAndLoadWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{
			Wantlists: []*pb.StoredWantlist{
				{
					Name: "testing",
				},
			},
		}}})

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" {
		t.Errorf("Bad list returned: %v", val)
	}

	// Also test that we can get all wantlists
	lists, err := s.ListWantlists(ctx, &pb.ListWantlistsRequest{})
	if err != nil {
		t.Fatalf("Error getting wantlists: %v", err)
	}
	if len(lists.GetLists()) != 1 || lists.GetLists()[0].GetName() != "testing" {
		t.Errorf("Bad wantlist return: %v", lists)
	}
}

func TestUpdateWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "testing",
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123},
				},
			},
		}},
	}})

	qc.FlushQueue(ctx)

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}
	log.Printf("GOT %v", val)

	if val.GetList().GetName() != "testing" {
		t.Fatalf("Bad list returned (name is wrong): %v", val)
	}

	if val.GetList().GetName() != "testing" || len(val.List.GetEntries()) != 1 || val.GetList().GetEntries()[0].GetId() != 123 {
		t.Fatalf("Bad list returned (name is wrong or entry is wrong): %v", val)
	}

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name:    "testing",
				Entries: []*pb.StoredWantlistEntry{},
			},
		}},
	}})

	val, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" || len(val.List.GetEntries()) != 0 {
		t.Errorf("Bad list returned (expected no entries): %v", val)
	}

}

func TestUpdateWantlist_NewType(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "testing",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123},
				},
			},
		}},
	}})

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" || val.GetList().GetType() != pb.WantlistType_EN_MASSE {
		t.Fatalf("Bad list returned initially: %v", val)
	}

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "testing",
				Type: pb.WantlistType_ONE_BY_ONE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123},
				},
			},
		}},
	}})
	val, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" ||
		val.GetList().GetType() != pb.WantlistType_ONE_BY_ONE {
		t.Errorf("Bad list returned (type should be 1b1): %v", val)
	}
}

func TestDeleteWantlist(t *testing.T) {
	ctx := getTestContext(123)
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Cannot save user: %v", err)
	}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := Server{d: d, di: di, qc: qc}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "testing",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123},
				},
			},
		}},
	}})

	lists, err := s.ListWantlists(ctx, &pb.ListWantlistsRequest{})
	if err != nil {
		t.Fatalf("unable to get wantlists: %v", err)
	}
	if len(lists.GetLists()) != 1 {
		t.Fatalf("Unable to get the list we just added: %v", lists)
	}

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig:     &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{}},
	}})
	lists, err = s.ListWantlists(ctx, &pb.ListWantlistsRequest{})
	if err != nil {
		t.Fatalf("unable to get wantlists: %v", err)
	}
	if len(lists.GetLists()) != 0 {
		t.Fatalf("Still finding the list we just added: %v", lists)
	}

}
