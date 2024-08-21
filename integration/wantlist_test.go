package integration

import (
	"log"
	"testing"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/server"

	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	rstore_client "github.com/brotherlogic/rstore/client"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestUpgradeToEnMasse(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name: "test-wantlist",
		Type: pb.WantlistType_ONE_BY_ONE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist")
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 124,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	for _, want := range wants.GetWants() {
		if want.GetWant().GetId() == 123 {
			if want.GetWant().GetState() != pb.WantState_WANTED {
				t.Errorf("First entry should be wanted: %v", want)
			}
		} else {
			if want.GetWant().GetState() == pb.WantState_WANTED {
				t.Errorf("Second entry should be wanted: %v", want)
			}
		}
	}

	s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:    "test-wantlist",
		NewType: pb.WantlistType_EN_MASSE,
	})

	qc.FlushQueue(ctx)

	wants, err = s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	for _, want := range wants.GetWants() {
		if want.GetWant().GetState() != pb.WantState_WANTED {
			t.Errorf("All entries should be wanted: %v", want)
		}
	}

}

func TestEnMasseWantlistUpdatedOnSync(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name: "test-wantlist",
		Type: pb.WantlistType_EN_MASSE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist")
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 124,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	for _, want := range wants.GetWants() {
		if want.GetWant().GetState() != pb.WantState_WANTED {
			t.Fatalf("All wants should be WANTED: %v", want)
		}
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWants{RefreshWants: &pb.RefreshWants{}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWantlists{RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})
	qc.FlushQueue(ctx)

	wants, err = s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get all the wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Fatalf("Should have 2 wants")
	}

	for _, want := range wants.GetWants() {
		if want.GetWant().GetId() == 123 {
			if want.GetWant().GetState() == pb.WantState_WANTED {
				t.Errorf("123 should be marked PURCHASED: %v", want)
			}
		} else {
			if want.GetWant().GetState() != pb.WantState_WANTED {
				t.Errorf("Want should be WANTED: %v", want)
			}
		}
	}

	list, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{
		Name: "test-wantlist",
	})

	for _, entry := range list.GetList().GetEntries() {
		if entry.GetId() == 123 {
			if entry.GetState() == pb.WantState_WANTED {
				t.Errorf("123 should be marked PURCHASED: %v", entry)
			}
		} else {
			if entry.GetState() != pb.WantState_WANTED {
				t.Errorf("Want should be WANTED: %v", entry)
			}
		}
	}
}

func TestWantlistScoreUpdatedOnSync(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name: "test-wantlist",
		Type: pb.WantlistType_ONE_BY_ONE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist")
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 124,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 || (wants.GetWants()[0].GetWant().Id != 123 && wants.GetWants()[1].GetWant().Id != 123) {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Rating: 5, Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWants{RefreshWants: &pb.RefreshWants{}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWantlists{RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})
	qc.FlushQueue(ctx)

	wants, err = s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get all the wants: %v", err)
	}

	// We should have wanted the new record
	found := false
	for _, r := range wants.GetWants() {
		if r.GetWant().GetId() == 124 {
			found = true
		}
	}

	if !found {
		t.Errorf("New want was not found: %v", err)
	}

	wl, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "test-wantlist"})
	if err != nil {
		t.Fatalf("Bad wantlist return: %v", err)
	}

	foundScore := false
	for _, entry := range wl.GetList().GetEntries() {
		if entry.GetScore() > 0 {
			foundScore = true
		}
	}

	if !foundScore {
		t.Errorf("Unable to find score: %v", wl)
	}

}


func TestWantlistUpdatedOnSync(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name: "test-wantlist",
		Type: pb.WantlistType_ONE_BY_ONE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist")
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist",
		AddId: 124,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 || (wants.GetWants()[0].GetWant().Id != 123 && wants.GetWants()[1].GetWant().Id != 123) {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWants{RefreshWants: &pb.RefreshWants{}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWantlists{RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})
	qc.FlushQueue(ctx)

	wants, err = s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get all the wants: %v", err)
	}

	// We should have wanted the new record
	found := false
	for _, r := range wants.GetWants() {
		if r.GetWant().GetId() == 124 {
			found = true
		}
	}

	if !found {
		t.Errorf("New want was not found: %v", err)
	}

}

func TestWantlistUpdatedOnSync_Hidden(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name:       "test-wantlist-visible",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_VISIBLE,
	})
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name:       "test-wantlist-invisible",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_INVISIBLE,
	})
	if err != nil {
		t.Fatalf("unable to add wantlist: %v", err)
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist-visible",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist-invisible",
		AddId: 124,
	})
	if err != nil {
		t.Fatalf("unable to add to wantlist: %v", err)
	}

	log.Printf("TEST sync 1")
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_SyncWants{SyncWants: &pb.SyncWants{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	log.Printf("WANTS: %v", wants)

	if len(wants.GetWants()) != 2 ||
		(wants.GetWants()[0].GetWant().Id != 123 && wants.GetWants()[1].GetWant().Id != 123) ||
		(wants.GetWants()[0].GetWant().State != pb.WantState_HIDDEN && wants.GetWants()[1].GetWant().State != pb.WantState_HIDDEN) {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}
}

func TestWantlistUpdatedOnSync_InvisibleAndHidden(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Create a want list
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name:       "test-wantlist-visible",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_VISIBLE,
	})
	_, err = s.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name:       "test-wantlist-invisible",
		Type:       pb.WantlistType_ONE_BY_ONE,
		Visibility: pb.WantlistVisibility_INVISIBLE,
	})
	if err != nil {
		t.Fatalf("Unable to add wantlist: %v", err)
	}

	// Update
	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist-visible",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	_, err = s.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
		Name:  "test-wantlist-invisible",
		AddId: 123,
	})
	if err != nil {
		t.Fatalf("Unable to add to wantlist: %v", err)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_SyncWants{SyncWants: &pb.SyncWants{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 1 ||
		wants.GetWants()[0].GetWant().Id != 123 ||
		wants.GetWants()[0].GetWant().State != pb.WantState_HIDDEN {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_SyncWants{SyncWants: &pb.SyncWants{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	dwants, _, err := di.GetWants(ctx, 1)

	if len(dwants) != 0 {
		t.Errorf("There should be only one non-hidden want: %v", dwants)
	}
}
