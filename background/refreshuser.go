package background

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/protobuf/proto"

	"github.com/brotherlogic/gramophile/db"

	pb "github.com/brotherlogic/gramophile/proto"
)

type BackgroundRunner struct {
	Database              *db.Database
	Key, Secret, Callback string
}

func (b *BackgroundRunner) RefreshUser(ctx context.Context, utoken, token, secret string) error {
	d := discogs.DiscogsWithToken(token, secret, b.Key, b.Secret, b.Callback)
	user, err := d.GetDiscogsUser(ctx)
	if err != nil {
		return err
	}

	log.Printf("GOT user: %v with %v %v %v", user, utoken, token, secret)

	su, err := b.Database.GetUser(ctx, utoken)
	if err != nil {
		return err
	}

	proto.Merge(su, &pb.StoredUser{User: user})
	su.LastRefreshTime = time.Now().Unix()

	return b.Database.SaveUser(ctx, su)
}
