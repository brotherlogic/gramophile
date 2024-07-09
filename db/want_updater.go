package db

import (
	"context"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

type WantChanger struct {
	queue pb.QueueServiceClient
}

func (w *WantChanger) Name() string {
	return "want_changer"
}

func (w *WantChanger) ProcessChange(ctx context.Context, c *pb.DBChange, config *pb.GramophileConfig) error {
	// We only care about this change if it's a change record
	if c.GetType() != pb.DBChange_CHANGE_WANT {
		return nil
	}

	if c.GetOldWant().GetState() != c.GetNewWant().GetState() {
		w.queue.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano() + int64(i),
				Entry: &pb.QueueElement_RefreshWant{
					RefreshWant: &pb.RefreshWant{Want: c.GetNewWant(), NewState: c.GetNewWant().GetState()},
				},
				Auth: entry.GetAuth(),
			},
		})
	}

}
