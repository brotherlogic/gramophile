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
	db db.Database
}

func (b *BackgroundRunner) RefreshUser(ctx context.Context, utoken, token, secret string) error {
	d := discogs.DiscogsWithToken(token, secret)
	user := d.GetDiscogsUser(ctx)

	su, err := b.db.GetUser(ctx, utoken)
	if err != nil {
		return err
	}

	proto.Merge(su.User, &pb.StoredUser{User: user})
	su.LastRefreshTime = time.Now().Unix()

	return b.db.SaveUser(ctx, su)
}
