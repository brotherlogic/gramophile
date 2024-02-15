package background

import (
	"context"
	"fmt"
	"testing"

	"github.com/brotherlogic/discogs"
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
	d := &discogs.TestDiscogsClient{}

	ctx := getTestContext(123)

	d.AddWant(ctx, 12345)

	b.AlignWants(ctx, d, &pb.WantsConfig{Origin: pb.WantsBasis_WANTS_GRAMOPHILE, Existing: pb.WantsExisting_EXISTING_DROP})

	t.Errorf("Finish this")

}
