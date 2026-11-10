package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestRefreshWant_HandlerRegistrationAndValidation(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	b.RegisterAllHandlers()

	entry := &pb.QueueElement{
		Auth: "test_auth_token",
		Entry: &pb.QueueElement_RefreshWant{
			RefreshWant: &pb.RefreshWant{
				Want: &pb.Want{
					Id: 12345,
				},
			},
		},
	}

	handler, err := b.getHandler(entry)
	if err != nil {
		t.Fatalf("handler not registered for QueueElement_RefreshWant: %v", err)
	}

	dedupKey := handler.GetDeduplicationKey(entry)
	if dedupKey != "RefreshWant-test_auth_token-12345" {
		t.Errorf("expected deduplication key 'RefreshWant-test_auth_token-12345', got '%v'", dedupKey)
	}

	err = handler.Validate(ctx, d, entry)
	if err != nil {
		t.Errorf("expected Validate to return nil, got %v", err)
	}
}

func TestRefreshWants_DoesNotDirtyCleanOwnedWant(t *testing.T) {
	ctx := context.Background()
	b := GetTestBackgroundRunner()
	di := &discogs.TestDiscogsClient{UserId: 123}

	// Seed an owned, arrived record with rating 4
	err := b.db.SaveRecord(ctx, 123, &pb.Record{
		Arrived: 1000,
		Release: &pbd.Release{Id: 101, InstanceId: 1001, Rating: 4},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("failed to save record 101: %v", err)
	}

	// Seed a second owned, arrived record whose want is still stuck at IN_TRANSIT
	err = b.db.SaveRecord(ctx, 123, &pb.Record{
		Arrived: 1000,
		Release: &pbd.Release{Id: 102, InstanceId: 1002, Rating: 5},
	}, &db.SaveOptions{})
	if err != nil {
		t.Fatalf("failed to save record 102: %v", err)
	}

	err = b.db.SaveWantlist(ctx, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, &pb.Wantlist{
		Name: "test-list",
		Entries: []*pb.WantlistEntry{
			{Id: 101, State: pb.WantState_PURCHASED},
			{Id: 102, State: pb.WantState_PURCHASED},
		},
	})
	if err != nil {
		t.Fatalf("failed to save wantlist: %v", err)
	}

	// Want 101 is already PURCHASED, Score=4, Clean=true
	err = b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            101,
		State:         pb.WantState_PURCHASED,
		IntendedState: pb.WantState_PURCHASED,
		Score:         4,
		Clean:         true,
		FromWantlist:  []string{"test-list"},
	}, "clean purchased want")
	if err != nil {
		t.Fatalf("failed to save want 101: %v", err)
	}

	// Want 102 has IntendedState=PURCHASED, Score=5, but State=IN_TRANSIT
	err = b.db.SaveWant(ctx, 123, &pb.Want{
		Id:            102,
		State:         pb.WantState_IN_TRANSIT,
		IntendedState: pb.WantState_PURCHASED,
		Score:         5,
		Clean:         false,
		FromWantlist:  []string{"test-list"},
	}, "stuck in-transit want")
	if err != nil {
		t.Fatalf("failed to save want 102: %v", err)
	}

	var enqueuedIDs []int64
	err = b.RefreshWants(ctx, di, "token-123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueuedIDs = append(enqueuedIDs, req.GetElement().GetRefreshWant().GetWant().GetId())
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("RefreshWants returned unexpected error: %v", err)
	}

	if len(enqueuedIDs) != 1 || enqueuedIDs[0] != 102 {
		t.Errorf("expected only want 102 to be enqueued, got %v", enqueuedIDs)
	}

	want101, err := b.db.GetWant(ctx, 123, 101)
	if err != nil {
		t.Fatalf("failed to load want 101: %v", err)
	}
	if !want101.GetClean() {
		t.Errorf("expected want 101 to remain Clean=true, got Clean=false")
	}
}
