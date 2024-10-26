package server

import (
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
			Origin:   pb.WantsBasis_WANTS_HYBRID}}})
	if err != nil {
		t.Fatalf("Bad config set: %v", err)
	}

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name:       "testing",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_INVISIBLE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "testing",
		AddId: 1234,
	})
	if err != nil {
		t.Fatalf("Unable to update wantlist: %v", err)
	}

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

	if wants.GetWants()[0].GetWant().State != pb.WantState_HIDDEN {
		t.Errorf("Want was not hidden: %v", wants)
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
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID}}})

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing", Type: pb.WantlistType_ONE_BY_ONE})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "testing",
		AddId: 1234,
	})
	if err != nil {
		t.Fatalf("Unable to update wantlist: %v", err)
	}

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
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID}}})

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

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
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID}}})

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("unable to add wantlist: %v", err)
	}

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" {
		t.Fatalf("Bad list returned: %v", val)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{Name: "testing", AddId: 123})
	if err != nil {
		t.Fatalf("Unable to update wantlist: %v", err)
	}

	val, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" || len(val.List.GetEntries()) != 1 || val.GetList().GetEntries()[0].GetId() != 123 {
		t.Errorf("Bad list returned: %v", val)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{Name: "testing", DeleteId: 123})
	if err != nil {
		t.Fatalf("Error updating wantlist: %v", err)
	}

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
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Existing: pb.WantsExisting_EXISTING_LIST, Origin: pb.WantsBasis_WANTS_HYBRID}}})

	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{Name: "testing", Type: pb.WantlistType_EN_MASSE})
	if err != nil {
		t.Fatalf("unable to add wantlist: %v", err)
	}

	val, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" || val.GetList().GetType() != pb.WantlistType_EN_MASSE {
		t.Fatalf("Bad list returned initially: %v", val)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{Name: "testing", NewType: pb.WantlistType_ONE_BY_ONE})
	if err != nil {
		t.Fatalf("Unable to update wantlist: %v", err)
	}

	val, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "testing"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	if val.GetList().GetName() != "testing" ||
		val.GetList().GetType() != pb.WantlistType_ONE_BY_ONE {
		t.Errorf("Bad list returned: %v", val)
	}
}
