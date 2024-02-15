package background

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/metadata"
)

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(),
		"auth-token",
		fmt.Sprintf("%v", userid))
}

func TestWantsDropped_Drop(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	c := &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_GRAMOPHILE}

	ctx := getTestContext(123)

	d.AddWant(ctx, 12345)
	_, err := b.PullWants(ctx, d, 1, 12345, c)
	if err != nil {
		t.Fatalf("Unable to pull wants: %v", err)
	}

	// The want should have been saved post pull
	wants, err := b.db.GetWants(ctx, 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 1 || wants[0].Id != 12345 {
		t.Errorf("Bad want pull: %v", wants)
	}

	b.AlignWants(ctx, d, c)

	// Want should have been dropped
	wants, err = b.db.GetWants(ctx, 123)
	if err != nil {
		t.Fatalf("Unable to load wants: %v", err)
	}

	if len(wants) != 0 {
		t.Errorf("Bad want pull post align: %v", wants)
	}

}
