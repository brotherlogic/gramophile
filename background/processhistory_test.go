package background

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestFanout(t *testing.T) {
	b := GetTestBackgroundRunner()
	b.db.SaveRecord(context.Background(), 123, &pb.Record{Release: &pbd.Release{Id: 1, DateAdded: 1234, Labels: []*pbd.Label{{Name: "testing"}}, InstanceId: 100}}, &db.SaveOptions{})

	fanoutCount := 0
	err := b.FanoutHistory(context.Background(), pb.UpdateType_UPDATE_WIDTH, &pb.StoredUser{
		Auth:    &pb.GramophileAuth{Token: "123"},
		User:    &pbd.User{DiscogsUserId: 123},
		Updates: &pb.UpdateControl{LastBackfill: make(map[string]int64)},
	}, "123", func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		fanoutCount++
		return &pb.EnqueueResponse{}, nil
	})

	if err != nil {
		t.Fatalf("Unable to fanout history: %v", err)
	}

	if fanoutCount != 1 {
		t.Errorf("Update was not fanned out")
	}
}
