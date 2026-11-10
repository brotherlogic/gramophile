package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSync_WithGramophile(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345}, "testing")

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}

	_, err = b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_GRAMOPHILE})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}
	err = b.CullWants(context.Background(), d, 12345)
	if err != nil {
		t.Fatalf("Unable to cull wants")
	}

	wants, err := b.db.GetWants(context.Background(), 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 1 || wants[0].Id != 12345 {
		t.Errorf("Wrong wants returned: %v", wants)
	}
}

func TestSync_WithDiscogs(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345}, "testing")

	d := &discogs.TestDiscogsClient{Wants: make(map[int64]*pbd.Want), UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	_, err = d.AddWant(context.Background(), 12346)
	if err != nil {
		t.Fatalf("Unable to add want: %v", err)
	}

	_, err = b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_DISCOGS})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}
	err = b.CullWants(context.Background(), d, 12345)
	if err != nil {
		t.Fatalf("Unable to cull wants")
	}

	wants, err := b.db.GetWants(context.Background(), 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 2 {
		t.Errorf("Wrong wants returned: %v", wants)
	}

	found := false
	for _, w := range wants {
		if w.GetId() == 12345 && w.State == pb.WantState_WANTED ||
			w.GetId() == 12346 && w.State != pb.WantState_WANTED {
			found = true
		}
	}

	if found {
		t.Errorf("Problem found: %v", wants)
	}
}

func TestSync_WithHybrid(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Errorf("Bad user save: %v", err)
	}

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345}, "testing")

	d := &discogs.TestDiscogsClient{Wants: make(map[int64]*pbd.Want), UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	_, err = d.AddWant(context.Background(), 12346)
	if err != nil {
		t.Fatalf("Unable to add want: %v", err)
	}

	_, err = b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}
	err = b.CullWants(context.Background(), d, 12345)
	if err != nil {
		t.Fatalf("Unable to cull wants")
	}

	wants, err := b.db.GetWants(context.Background(), 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 2 {
		t.Errorf("Wrong wants returned: %v", wants)
	}
}

func TestSyncWants_EnqueuesHaveIntention(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(context.Background(), &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	// Seed an unclean want
	err = b.db.SaveWant(context.Background(), 123, &pb.Want{
		Id:            12345,
		State:         pb.WantState_IN_TRANSIT,
		IntendedState: pb.WantState_IN_TRANSIT,
		Clean:         false,
	}, "testing unclean want")
	if err != nil {
		t.Fatalf("Bad want save: %v", err)
	}

	d := &discogs.TestDiscogsClient{Wants: make(map[int64]*pbd.Want), UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}

	var enqueued []*pb.EnqueueRequest
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueued = append(enqueued, req)
		return &pb.EnqueueResponse{}, nil
	}

	user, err := b.db.GetUser(context.Background(), "123")
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}

	entry := &pb.QueueElement{
		Intention: "From Test",
		Auth:      "123",
		Entry: &pb.QueueElement_SyncWants{
			SyncWants: &pb.SyncWants{Page: 1},
		},
	}

	err = b.ProcessSyncWants(context.Background(), d, user, entry, enqueue)
	if err != nil {
		t.Fatalf("ProcessSyncWants failed: %v", err)
	}

	if len(enqueued) == 0 {
		t.Fatalf("Expected enqueued requests, got 0")
	}

	for _, req := range enqueued {
		if req.GetElement().GetIntention() == "" {
			t.Errorf("Enqueued element missing intention: %v", req.GetElement())
		}
	}
}

func TestSyncWants_EnqueuesMismatchedStateAndPrioritizesWanted(t *testing.T) {
	ctx := context.Background()
	b := GetTestBackgroundRunner()

	err := b.db.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Bad user save: %v", err)
	}

	// Seed a want stuck with State=RETIRED, IntendedState=PENDING, Clean=true (lower ID)
	err = b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            100,
		State:         pb.WantState_RETIRED,
		IntendedState: pb.WantState_PENDING,
		Clean:         true,
	}, "stuck pending want")
	if err != nil {
		t.Fatalf("Bad want save: %v", err)
	}

	// Seed a want stuck with State=PENDING, IntendedState=WANTED, Clean=true (higher ID)
	err = b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            200,
		State:         pb.WantState_PENDING,
		IntendedState: pb.WantState_WANTED,
		Clean:         true,
	}, "stuck wanted want")
	if err != nil {
		t.Fatalf("Bad want save: %v", err)
	}

	d := &discogs.TestDiscogsClient{Wants: make(map[int64]*pbd.Want), UserId: 123}

	var enqueuedWantIDs []int64
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		if rw := req.GetElement().GetRefreshWant(); rw != nil {
			enqueuedWantIDs = append(enqueuedWantIDs, rw.GetWant().GetId())
		}
		return &pb.EnqueueResponse{}, nil
	}

	user, err := b.db.GetUser(ctx, "123")
	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}

	entry := &pb.QueueElement{
		Intention: "From Test",
		Auth:      "123",
		Entry: &pb.QueueElement_SyncWants{
			SyncWants: &pb.SyncWants{Page: 1},
		},
	}

	if err := b.ProcessSyncWants(ctx, d, user, entry, enqueue); err != nil {
		t.Fatalf("ProcessSyncWants failed: %v", err)
	}

	if len(enqueuedWantIDs) != 2 || enqueuedWantIDs[0] != 200 || enqueuedWantIDs[1] != 100 {
		t.Errorf("expected both wants to be enqueued with WANTED want 200 first, got %v", enqueuedWantIDs)
	}
}
