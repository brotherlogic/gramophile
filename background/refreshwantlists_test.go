package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
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
		Name: "digital_wantlist",
		Entries: []*pb.WantlistEntry{
			{Id: 0},
			{Id: 12},
		}}

	err = b.processWantlist(context.Background(), di, &pb.WantslistConfig{}, wl, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
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

	err = b.processWantlist(context.Background(), di,
		&pb.WantslistConfig{
			MinScore: 2.5,
			MinCount: 0,
		}, wl, "123", func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
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
