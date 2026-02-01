package integration

import (
	"log"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestCreateDigitalWantlist(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2, Formats: []*pbd.Format{{Name: "Vinyl"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "File"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	if len(r.GetRecords()[0].GetRecord().GetDigitalVersions()) != 2 {
		t.Fatalf("Record has too many digitial versions: %v", r)
	}

	// It should have no entries
	if len(wl.GetList().GetEntries()) != 0 {
		t.Errorf("Wanlist has  been populated too early: %v", wl)
	}

	// Setting the release to be DIGITAL_WANTED
	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_DIGITAL_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	qc.FlushQueue(ctx)

	// We should have a digital wantslist
	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	// It should have two entries now
	if len(wl.GetList().GetEntries()) != 2 {
		t.Errorf("Wanlist has not been populated: %v (%v)", wl, len(wl.GetList().GetEntries()))
	}
}

func TestRemoveFromDigitalWantlist(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "File"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_DIGITAL_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 2 {
		t.Errorf("Wanlist has not been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	for _, entry := range wl.GetList().GetEntries() {
		if entry.GetState() != pb.WantState_WANTED {
			want, err := d.GetWant(ctx, 123, entry.GetId())
			t.Fatalf("Wantlist should all be WANTED: %v; %v -> %v", entry, want, err)
		}
	}

	/*
	************************************
	 */

	// Now purchase one of the entries
	_, err = s.AddRecord(ctx, &pb.AddRecordRequest{
		Id:       4,
		Location: "Direct",
		Price:    123,
	})
	if err != nil {
		t.Errorf("Unable to add want: %v", err)
	}

	// Digital wantlist should have no active entries (post flush)
	qc.FlushQueue(ctx)
	log.Printf("FLISHED THE QUEUE")

	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error in getting wantlist: %v", err)
	}

	for _, entry := range wl.GetList().GetEntries() {
		if entry.GetState() == pb.WantState_WANTED {
			want, err := d.GetWant(ctx, 123, entry.GetId())
			t.Errorf("%v is still WANTED, it should be BOUGHT (%v / %v is the want)", entry, want, err)
		}
	}

	wants, err := d.GetWants(ctx, 123)
	if err != nil {
		t.Errorf("Bad get wants: %v", err)
	}
	log.Printf("Found %v wants", len(wants))
	for _, want := range wants {
		log.Printf("WANT: %v", want)
	}
}

func TestChangeKeepRemoveDigitalWantlist(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "Vinyl"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_DIGITAL_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 1 {
		t.Errorf("Wanlist has not been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	// Manually adding an entry to the digital wantlist
	wantlist, err := d.LoadWantlist(ctx, 123, "digital_wantlist")
	if err != nil {
		t.Fatalf("Unable to load wantlist: %v", err)
	}
	wantlist.Entries = append(wantlist.Entries, &pb.WantlistEntry{
		Id:    200,
		State: pb.WantState_WANTED})
	d.SaveWantlist(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, wantlist)

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   12345,
			Entry:     &pb.QueueElement_RefreshWantlists{},
		},
	})
	qc.FlushQueue(ctx)

	// We should have a digital wantslist
	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	// It should have one entry now as the 200 id
	if len(wl.GetList().GetEntries()) != 1 {
		t.Errorf("Wanlist has not been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}
}

func TestBadAddToDigitalWantlist(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "Vinyl"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_DIGITAL_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 1 {
		t.Errorf("Wanlist has not been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	// Reset the want status
	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_NO_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

	// We should have a digital wantslist
	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}

	// It should have no entries
	if len(wl.GetList().GetEntries()) != 0 {
		t.Errorf("Wanlist still has entries: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}
}

func TestWantlistCleanoutCorrect(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "Vinyl"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_NO_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 0 {
		t.Errorf("Wanlist has  been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	// Manually add the want to the digital wantlist
	list, err := d.LoadWantlist(ctx, 123, "digital_wantlist")
	if err != nil {
		t.Fatalf("Unable to load wantlist: %v", err)
	}
	list.Entries = append(list.GetEntries(), &pb.WantlistEntry{
		Id:       3,
		SourceId: 1,
		State:    pb.WantState_WANTED})
	err = d.SaveWantlist(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, list)
	if err != nil {
		t.Errorf("Unable to save wantlist: %v", err)
	}

	// We should have a digital wantslist
	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}
	// It should have one entry
	if len(wl.GetList().GetEntries()) != 1 {
		t.Errorf("Wanlist has not  been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	// Now run a wantlist sync
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   12345,
			Entry:     &pb.QueueElement_RefreshWantlists{},
		},
	})
	qc.FlushQueue(ctx)

	// We should have a digital wantslist
	wl, err = s.GetWantlist(ctx, &pb.GetWantlistRequest{Name: "digital_wantlist"})
	if err != nil {
		t.Fatalf("Error getting wantlist: %v", err)
	}
	// It should have one entry
	if len(wl.GetList().GetEntries()) != 0 {
		t.Errorf("Wanlist is still populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}
}

func TestWantRemovedOnWantlistRemoval(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	di.AddCollectionRelease(&pbd.Release{MasterId: 200, Id: 1, InstanceId: 100, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 2, Rating: 2})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 3, Rating: 2, Formats: []*pbd.Format{{Name: "CD"}}})
	di.AddNonCollectionRelease(&pbd.Release{MasterId: 200, Id: 4, Rating: 2, Formats: []*pbd.Format{{Name: "Vinyl"}}})

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	qc.FlushQueue(ctx)

	_, err = s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: 100,
		Intent:     &pb.Intent{Keep: pb.KeepStatus_NO_KEEP},
	})
	if err != nil {
		t.Fatalf("Unable to set intent: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry:     &pb.QueueElement_RefreshCollection{},
		},
	})

	qc.FlushQueue(ctx)

	log.Printf("FINISHED FLUSH")

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
	if len(r.GetRecords()[0].GetRecord().GetDigitalIds()) == 0 {
		t.Fatalf("Record has no digital versions: %v", r)
	}

	// It should have one entry
	if len(wl.GetList().GetEntries()) != 0 {
		t.Errorf("Wanlist has  been populated: (%v) %v", len(wl.GetList().GetEntries()), wl)
	}

	// Manually add a new want
	err = d.SaveWant(ctx, 123, &pb.Want{
		Id:            3,
		FromWantlist:  []string{"digital_wantlist"},
		State:         pb.WantState_WANTED,
		IntendedState: pb.WantState_WANTED,
	}, "creating")
	if err != nil {
		t.Fatalf("Unable to load wantlist: %v", err)
	}
	// Now run a want sync
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   12345,
			Entry:     &pb.QueueElement_RefreshWants{},
		},
	})
	qc.FlushQueue(ctx)

	wants, err := s.GetWants(ctx, &pb.GetWantsRequest{})
	if err != nil {
		t.Fatalf("Unable to get wants")
	}

	if len(wants.GetWants()) != 1 {
		t.Errorf("Wants were not preserved: %v", wants)
	}

	if wants.GetWants()[0].GetWant().GetState() == pb.WantState_WANTED {
		t.Errorf("Bad resultant want: %v", wants)
	}
}

func TestHangingWantRemoved(t *testing.T) {
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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 15, Name: "Keep"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
	qc := queuelogic.GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err = s.SetConfig(ctx, &pb.SetConfigRequest{
		Config: &pb.GramophileConfig{
			WantsConfig: &pb.WantsConfig{
				DigitalWantList: true,
			},
			AddConfig: &pb.AddConfig{
				Adds:          pb.Enabled_ENABLED_ENABLED,
				DefaultFolder: "12 Inches",
			},
		},
	})
	if err != nil {
		t.Fatalf("Unable to set config: %v", err)
	}

	// Let's create a haning want
	err = d.SaveWant(ctx, 123, &pb.Want{
		Id:           3,
		FromWantlist: []string{"digital_wantlist"},
	}, "Adding for test")
	if err != nil {
		t.Fatalf("Unable to save want: %v", err)
	}

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			RunDate:   time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{}},
		},
	})

	qc.FlushQueue(ctx)

	// If we load the want, it should have the digital_wantlist chip removed
	want, err := d.GetWant(ctx, 123, 3)
	if err != nil {
		t.Fatalf("Unable to load want: %v", err)
	}

	found := false
	for _, list := range want.GetFromWantlist() {
		if list == "digital_wantlist" {
			found = true
		}
	}

	if found {
		t.Errorf("Want is still part of the digital wantlist: %v", want)
	}

}
