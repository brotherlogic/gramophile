package integration

import (
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestDowngradeToOneByOne(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Fatalf("Bad wants returned first pass (expected to see original want): %v", wants)
	}

	for _, want := range wants.GetWants() {
		if want.GetWant().GetId() == 123 {
			if want.GetWant().GetState() != pb.WantState_WANTED {
				t.Errorf("First entry should be wanted: %v", want)
			}
		} else {
			if want.GetWant().GetState() != pb.WantState_WANTED {
				t.Errorf("Second entry should  be wanted: %v", want)
			}
		}
	}
	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_ONE_BY_ONE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

	qc.FlushQueue(ctx)

	wants, err = s.GetWants(ctx, &pb.GetWantsRequest{})
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
				t.Errorf("Second entry should  not be wanted: %v", want)
			}
		}
	}

}

func TestUpgradeToEnMasse(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_ONE_BY_ONE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

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
				t.Errorf("PrU First entry should be wanted: %v", want)
			}
		} else {
			if want.GetWant().GetState() == pb.WantState_WANTED {
				t.Errorf("PrU Second entry should not be wanted: %v", want)
			}
		}
	}

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

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

func TestWantlistLifecycle(t *testing.T) {

	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 1234},
				},
			},
		}}}})

	err = qc.FlushQueue(ctx)
	if err != nil {
		t.Fatalf("Unable to flush queue: %v", err)
	}

	wantlist, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "test-wantlist"})
	if err != nil {
		t.Fatalf("Unable to get wantlist: %v", err)
	}

	if wantlist.GetList().GetEntries()[0].GetState() != pb.WantState_WANTED {
		t.Errorf("Want was not wanted: %v", wantlist.GetList().GetEntries()[0])
	}

	// purchase want
	di.AddCollectionRelease(&pbd.Release{Id: 1234, InstanceId: 12345})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 1,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshCollectionEntry{RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWants{RefreshWants: &pb.RefreshWants{}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 3,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWantlists{RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})
	qc.FlushQueue(ctx)

	wantlist, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "test-wantlist"})
	if err != nil {
		t.Fatalf("Unable to get wantlist: %v", err)
	}

	if wantlist.GetList().GetEntries()[0].GetState() != pb.WantState_IN_TRANSIT {
		t.Errorf("Want was not IN TRANSIT: %v", wantlist)
	}

	// Record has arrived
	s.SetIntent(ctx, &pb.SetIntentRequest{InstanceId: 12345, Intent: &pb.Intent{Arrived: time.Now().UnixNano()}})
	qc.FlushQueue(ctx)

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 2,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWants{RefreshWants: &pb.RefreshWants{}},
		},
	})

	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 3,
			Auth:    "123",
			Entry:   &pb.QueueElement_RefreshWantlists{RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})
	qc.FlushQueue(ctx)

	wantlist, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "test-wantlist"})
	if err != nil {
		t.Fatalf("Unable to get wantlist: %v", err)
	}

	if wantlist.GetList().GetEntries()[0].GetState() != pb.WantState_PURCHASED {
		t.Errorf("Want was not wanted: %v", wantlist)
	}
}
func TestEnMasseWantlistUpdatedOnSync(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

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

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{})
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

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 || (wants.GetWants()[0].GetWant().Id != 123 && wants.GetWants()[1].GetWant().Id != 123) {
		t.Fatalf("Bad wants returned (expected to see original want): %v", wants)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{Rating: 5, Id: 123, InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}}, &db.SaveOptions{})
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

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	s.SetConfig(ctx, &pb.SetConfigRequest{Config: &pb.GramophileConfig{
		WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID},
		WantsListConfig: &pb.WantslistConfig{Wantlists: []*pb.StoredWantlist{
			{
				Name: "test-wantlist",
				Type: pb.WantlistType_EN_MASSE,
				Entries: []*pb.StoredWantlistEntry{
					{Id: 123}, {Id: 124}},
			},
		},
		},
	}})

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

func TestBuildDigitalWantlist(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 20, Name: "Keep"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			KeepConfig: &pb.KeepConfig{
				Enabled: pb.Enabled_ENABLED_ENABLED,
			},
		},
	})
	if err != nil {
		t.Fatalf("Cannot set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent: &pb.Intent{
			Keep: pb.KeepStatus_DIGITAL_KEEP,
		},
	})
	if err != nil {
		t.Fatalf("Bad intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry:   &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	// We should have a digital wantslist
	wl, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	// Our record should have digital versions
	r, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{GetRecordWithId: &pb.GetRecordWithId{InstanceId: 100}}})
	if err != nil {
		t.Fatalf("Cannot get record: %v", err)
	}
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) != 1 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 1 {
		t.Fatalf("Wanlist has not been populated: %v", wl)
	}

	if wl.GetList().GetEntries()[0].GetId() != r.GetRecords()[0].GetRecord().GetDigitalIds()[0] {
		t.Errorf("Mismatch in wants")
	}
}

func TestBuildMintUplWantlist(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Keep"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})
	qc.FlushQueue(ctx)

	s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				MintUpWantList: true,
			},
		},
	})

	// Set the record to keep_mint
	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent: &pb.Intent{
			Keep:    pb.KeepStatus_MINT_UP_KEEP,
			MintIds: []int64{12},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	qc.FlushQueue(ctx)

	// We should have a digital wantslist
	wl, err := s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "mint_up_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 1 {
		t.Errorf("Wanlist has not been populated: %v", wl)
	}
}

func TestWantlistDisabledOnListening(t *testing.T) {
	ctx := getTestContext(123)

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Can't init save user: %v", err)
	}
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	// Add releases to go over the threshold
	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 5, FolderId: 12}})
	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{MasterId: 200, Id: 2, InstanceId: 101, Rating: 5, FolderId: 12}})
	d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{MasterId: 200, Id: 3, InstanceId: 102, Rating: 5, FolderId: 12}})

	s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			OrganisationConfig: &pb.OrganisationConfig{
				Organisations: []*pb.Organisation{
					{
						Name:       "Test",
						Use:        pb.OrganisationUse_ORG_USE_LISTENING,
						Foldersets: []*pb.FolderSet{{Name: "Listening Pile", Folder: 12}},
					},
				},
			},
			WantsListConfig: &pb.WantslistConfig{ListeningThreshold: 2, Wantlists: []*pb.StoredWantlist{
				{
					Name: "test-wantlist-visible",
					Type: pb.WantlistType_ONE_BY_ONE,
					Entries: []*pb.StoredWantlistEntry{
						{Id: 123}, {Id: 124}},
				}}},
			WantsConfig: &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID}}})

	// Validate that the org has records in it
	org, err := s.GetOrg(ctx, &pb.GetOrgRequest{OrgName: "Test"})
	if err != nil {
		t.Fatalf("Unable to get org: %v", err)
	}
	if len(org.GetSnapshot().GetPlacements()) != 3 {
		t.Fatalf("Records were not placed in the org: %v", org)
	}

	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants: %v", err)
	}

	if len(wants.GetWants()) != 2 {
		t.Errorf("Wants not created: %v", wants)
	}

	for _, w := range wants.GetWants() {
		if w.GetWant().GetState() == pb.WantState_WANTED {
			t.Errorf("Want %v is wanted -> should not be", w)
		}
	}
}
