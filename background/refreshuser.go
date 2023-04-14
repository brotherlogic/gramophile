package background

import (
	"context"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/protobuf/proto"

	"github.com/brotherlogic/gramophile/db"

	pb "github.com/brotherlogic/gramophile/proto"
)

type BackgroundRunner struct {
	db                    *db.Database
	user                  *pb.StoredUser
	Key, Secret, Callback string
	d                     *discogs.Discogs
}

func (b *BackgroundRunner) RefreshUser(ctx context.Context, utoken string) error {
	user, err := b.d.GetDiscogsUser(ctx)
	if err != nil {
		return err
	}

	su, err := b.db.GetUser(ctx, utoken)
	if err != nil {
		return err
	}

	proto.Merge(su, &pb.StoredUser{User: user})
	su.LastRefreshTime = time.Now().Unix()

	return b.db.SaveUser(ctx, su)
}
