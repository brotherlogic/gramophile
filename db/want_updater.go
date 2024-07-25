package db

import (
	"context"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

type WantChanger struct {
	queue pb.QueueServiceClient
}

func (w *WantChanger) Name() string {
	return "want_changer"
}

func (w *WantChanger) ProcessChange(ctx context.Context, c *pb.DBChange, user *pb.StoredUser) error {
	log.Printf("CHANGE %v", c)
	// We only care about this change if it's a change record
	if c.GetType() != pb.DBChange_CHANGE_WANT {
		return nil
	}

	if c.GetOldWant().GetState() != c.GetNewWant().GetState() {
		_, err := w.queue.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_RefreshWant{
					RefreshWant: &pb.RefreshWant{Want: c.GetNewWant(), NewState: c.GetNewWant().GetState()},
				},
				Auth: user.GetAuth().GetToken(),
			},
		})
		return err
	}

	return nil
}
