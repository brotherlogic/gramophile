package background

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/protobuf/proto"

	pb "github.com/brotherlogic/gramophile/proto"
)

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

	// Validate
	for _, typ := range []pb.UpdateType{
		pb.UpdateType_UPDATE_FOLDER,
		pb.UpdateType_UPDATE_GOAL_FOLDER,
		pb.UpdateType_UPDATE_WIDTH,
	} {
		if su.GetUpdates().GetLastBackfill()[typ.String()] == 0 {
			_, err := enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Auth:    "123",
					RunDate: time.Now().UnixNano(),
					Entry: &pb.QueueElement_FanoutHistory{
						FanoutHistory: &pb.FanoutHistory{
							Userid: int64(123),
							Type:   pb.UpdateType_UPDATE_WIDTH,
						},
					},
				},
			})
			if err != nil {
				return err
			}
		}
	}

	return b.db.SaveUser(ctx, su)
}
