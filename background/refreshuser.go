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
	db                    db.Database
	key, secret, callback string
}

func GetBackgroundRunner(db db.Database, key, secret, callback string) *BackgroundRunner {
	return &BackgroundRunner{db: db, key: key, secret: secret, callback: callback}
}

func (b *BackgroundRunner) RefreshUser(ctx context.Context, d discogs.Discogs, utoken string) error {
	user, err := d.GetDiscogsUser(ctx)
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
