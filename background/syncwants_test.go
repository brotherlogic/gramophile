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

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345})

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}

	_, err := b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_GRAMOPHILE})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
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

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345})

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}

	_, err := b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_DISCOGS})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}

	wants, err := b.db.GetWants(context.Background(), 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 1 || wants[0].Id != 12346 {
		t.Errorf("Wrong wants returned: %v", wants)
	}
}

func TestSync_WithHybrid(t *testing.T) {
	b := GetTestBackgroundRunner()

	// Seed a saved want
	b.db.SaveWant(context.Background(), 123, &pb.Want{Id: 12345})

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}

	_, err := b.PullWants(context.Background(), d, 1, 12345, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_HYBRID})
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}

	wants, err := b.db.GetWants(context.Background(), 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 2 {
		t.Errorf("Wrong wants returned: %v", wants)
	}
}
