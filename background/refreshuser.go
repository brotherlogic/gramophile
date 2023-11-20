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

	folders, err := d.GetUserFolders(ctx)
	log.Printf("got user folders %v and %v", folders, err)
	if err != nil {
		return err
	}
	su.Folders = folders
	su.LastRefreshTime = time.Now().UnixNano()

	return b.db.SaveUser(ctx, su)
}
