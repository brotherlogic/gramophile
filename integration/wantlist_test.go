package integration

import (
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

	if len(wants.GetWants()) != 2 || wants.GetWants()[0].Id != 123 {
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
		if r.GetId() == 124 {
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
		AddId: 124,
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

	if len(wants.GetWants()) != 2 ||
		(wants.GetWants()[0].Id != 123 && wants.GetWants()[1].Id != 123) ||
		(wants.GetWants()[0].State != pb.WantState_HIDDEN && wants.GetWants()[1].State != pb.WantState_HIDDEN) {
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

	if len(dwants) != 1 {
		t.Errorf("There should be only one non-hidden want: %v", dwants)
	}
}

func TestWantlistUpdatedOnSync_HiddenAndInvisible(t *testing.T) {
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
		wants.GetWants()[0].Id != 123 ||
		wants.GetWants()[0].State != pb.WantState_WANTED {
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

	if len(dwants) != 1 {
		t.Errorf("There should be only one non-hidden want: %v", dwants)
	}
}
