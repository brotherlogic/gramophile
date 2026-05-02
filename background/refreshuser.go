package background

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	"google.golang.org/protobuf/proto"

	pb "github.com/brotherlogic/gramophile/proto"
)

type refreshUserHandler struct {
	b *BackgroundRunner
}

func (h *refreshUserHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth(), enqueue)
}

func (h *refreshUserHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *refreshUserHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) RefreshUser(ctx context.Context, d discogs.Discogs, utoken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
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
