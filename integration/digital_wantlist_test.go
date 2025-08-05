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
				AllowAdds:     pb.Mandate_REQUIRED,
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
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

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
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Arrived"}, {Id: 20, Name: "Purchase Price"}, {Id: 30, Name: "Purchase Location"}}}
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
				AllowAdds:     pb.Mandate_REQUIRED,
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
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}},
		},
	})

	// Queue up a collection refresh
	qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Auth:    "123",
			RunDate: time.Now().UnixNano(),
			Entry:   &pb.QueueElement_RefreshCollection{},
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
		t.Errorf("Wanlist has not been populated: %v", wl)
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
