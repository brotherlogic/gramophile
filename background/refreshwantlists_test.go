package background

import (
	"context"
	"log"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestZeroEntriesInWantlist(t *testing.T) {
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	wl := &pb.Wantlist{
		Name: "test_clearing_list",
		Type: pb.WantlistType_EN_MASSE,
		Entries: []*pb.WantlistEntry{
			{Id: 0},
			{Id: 12},
		}}

	err = b.processWantlist(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, di, &pb.WantslistConfig{}, wl, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("Unable to process wantlist: %v", err)
	}

	if len(wl.GetEntries()) != 1 {
		t.Errorf("Bad want was not cleared: %v", wl)
	}
}

func TestInactiveWantlist(t *testing.T) {
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	wl := &pb.Wantlist{
		Name: "test-wantlist",
		Entries: []*pb.WantlistEntry{
			{Id: 5},
			{Id: 12},
		}}
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 5, Score: 2, State: pb.WantState_PURCHASED}, "testing")

	err = b.processWantlist(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, di,
		&pb.WantslistConfig{
			MinScore: 2.5,
			MinCount: 0,
		}, wl, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
			return &pb.EnqueueResponse{}, nil
		})

	list, err := b.db.LoadWantlist(context.Background(), 123, "test-wantlist")
	if err != nil {
		t.Fatalf("Unable to load wantlist: %v", err)
	}

	for _, entry := range list.GetEntries() {
		if entry.GetState() != pb.WantState_RETIRED {
			t.Errorf("Want should be retired: %v", entry)
		}
	}

}

func TestTimedWantlist(t *testing.T) {

	var testCases = []struct {
		name          string
		wantlist      *pb.Wantlist
		expectedCount int
	}{
		{
			"Not Started",
			&pb.Wantlist{
				Name:      "timed_wantlist",
				Type:      pb.WantlistType_DATE_BOUNDED,
				StartDate: time.Now().Add(time.Hour * 24).UnixNano(),
				EndDate:   time.Now().Add(time.Hour * 24 * 2).UnixNano(),
				Entries: []*pb.WantlistEntry{
					{Id: 1},
					{Id: 2},
					{Id: 3},
				}},
			0,
		},
		{
			"Day 1",
			&pb.Wantlist{
				Name:      "timed_wantlist",
				Active:    true,
				Type:      pb.WantlistType_DATE_BOUNDED,
				StartDate: time.Now().UnixNano(),
				EndDate:   time.Now().Add(time.Hour * 24 * 3).UnixNano(),
				Entries: []*pb.WantlistEntry{
					{Id: 1},
					{Id: 2},
					{Id: 3},
				}},
			1,
		},
		{
			"Day 2",
			&pb.Wantlist{
				Name:      "timed_wantlist",
				Active:    true,
				Type:      pb.WantlistType_DATE_BOUNDED,
				StartDate: time.Now().Add(-time.Hour * 24).UnixNano(),
				EndDate:   time.Now().Add(time.Hour * 24 * 2).UnixNano(),
				Entries: []*pb.WantlistEntry{
					{Id: 1},
					{Id: 2},
					{Id: 3},
				}},
			2,
		},
		{
			"Day 3",
			&pb.Wantlist{
				Name:      "timed_wantlist",
				Active:    true,
				Type:      pb.WantlistType_DATE_BOUNDED,
				StartDate: time.Now().Add(-time.Hour * 24 * 2).UnixNano(),
				EndDate:   time.Now().UnixNano(),
				Entries: []*pb.WantlistEntry{
					{Id: 1},
					{Id: 2},
					{Id: 3},
				}},
			3,
		},
	}

	for _, tc := range testCases {
		b := GetTestBackgroundRunner()
		di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

		err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
		if err != nil {
			t.Errorf("Bad user save: %v", err)
		}

		counted := 0
		log.Printf("PROCESSING")
		err = b.processWantlist(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, di, &pb.WantslistConfig{}, tc.wantlist, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
			if req.GetElement().GetRefreshWant().GetWant().GetState() == pb.WantState_WANTED {
				counted++
				log.Printf("COUNTED %v", req)
			}
			return &pb.EnqueueResponse{}, nil
		})

		log.Printf("REFRESHING")
		_, err = b.refreshWantlist(context.Background(), 123, tc.wantlist, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
			counted++
			log.Printf("ENQUEUING INTERNAL TEST")
			return &pb.EnqueueResponse{}, nil
		})
		if err != nil {
			t.Fatalf("Unable to process wantlist: %v", err)
		}

		if counted != tc.expectedCount {
			t.Errorf("Wrong number of additions to the wantlist: %v  -> %v (for %v)", counted, tc.expectedCount, tc.name)
		}
	}
}

func TestFastPathPurchased(t *testing.T) {
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Goal Folder"}}}

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	// Seed a record as arrived
	b.db.SaveRecord(context.Background(), 123, &pb.Record{
		Arrived: 12345,
		Release: &pbd.Release{Id: 730165, InstanceId: 100},
	}, &db.SaveOptions{})

	wl := &pb.Wantlist{
		Name: "test-wantlist",
		Type: pb.WantlistType_ONE_BY_ONE,
		Entries: []*pb.WantlistEntry{
			{Id: 730165, State: pb.WantState_WANTED},
		}}

	// Seed the want as WANTED (so the fast-path is needed to override)
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 730165, State: pb.WantState_WANTED}, "testing")

	err = b.processWantlist(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, di, &pb.WantslistConfig{}, wl, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("Unable to process wantlist: %v", err)
	}

	if wl.GetEntries()[0].GetState() != pb.WantState_PURCHASED {
		t.Errorf("Fast-path failed: entry state is %v, expected PURCHASED", wl.GetEntries()[0].GetState())
	}
}

func TestRefreshWantlistsDeduplicationKey(t *testing.T) {
	b := GetTestBackgroundRunner()
	handler := &refreshWantlistsHandler{b: b}

	entry := &pb.QueueElement{
		Auth: "user-token-123",
		Entry: &pb.QueueElement_RefreshWantlists{
			RefreshWantlists: &pb.RefreshWantlists{},
		},
	}

	key := handler.GetDeduplicationKey(entry)
	expected := "RefreshWantlists-user-token-123"
	if key != expected {
		t.Errorf("Unexpected deduplication key: got %q, want %q", key, expected)
	}
}

func TestOneByOneWantlist_ReenqueuesStuckWantedEntryAndMarksDirty(t *testing.T) {
	ctx := context.Background()
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123}

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}
	if err := b.db.SaveUser(ctx, user); err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	// Simulate a want where mergeWant previously set IntendedState=WANTED while State=PENDING and Clean=true,
	// and the RefreshWant task was dropped due to MaxUserQueueSize throttling.
	err := b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            444484,
		State:         pb.WantState_PENDING,
		IntendedState: pb.WantState_WANTED,
		Clean:         true,
		FromWantlist:  []string{"Analord"},
	}, "stuck wanted entry")
	if err != nil {
		t.Fatalf("Bad want save: %v", err)
	}

	// Also simulate a subsequent entry stuck at State=RETIRED, IntendedState=PENDING, Clean=true.
	err = b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            444485,
		State:         pb.WantState_RETIRED,
		IntendedState: pb.WantState_PENDING,
		Clean:         true,
		FromWantlist:  []string{"Analord"},
	}, "stuck pending entry")
	if err != nil {
		t.Fatalf("Bad want save: %v", err)
	}

	wl := &pb.Wantlist{
		Name:   "Analord",
		Type:   pb.WantlistType_ONE_BY_ONE,
		Active: true,
		Entries: []*pb.WantlistEntry{
			{Id: 444484, Index: 1, State: pb.WantState_WANTED},
			{Id: 444485, Index: 2, State: pb.WantState_PENDING},
		},
	}

	var enqueuedIDs []int64
	err = b.processWantlist(ctx, user, di, &pb.WantslistConfig{}, wl, "123", false, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueuedIDs = append(enqueuedIDs, req.GetElement().GetRefreshWant().GetWant().GetId())
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("processWantlist failed: %v", err)
	}

	if len(enqueuedIDs) != 2 || enqueuedIDs[0] != 444484 || enqueuedIDs[1] != 444485 {
		t.Errorf("expected both 444484 and 444485 to be re-enqueued, got %v", enqueuedIDs)
	}

	for _, id := range []int64{444484, 444485} {
		w, err := b.db.GetWant(ctx, 123, id)
		if err != nil {
			t.Fatalf("failed to load want %d: %v", id, err)
		}
		if w.GetClean() {
			t.Errorf("expected want %d to have Clean=false, got Clean=true", id)
		}
	}
}

