package db

import (
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func buildWantUpdates(old, new *pb.Want) *pb.Update {
	update := &pb.Update{Date: time.Now().UnixNano()}

	if old == nil {
		update.Changes = append(update.Changes, &pb.Change{
			Type:        pb.Change_ADDED,
			Description: "Want created",
		})
		return update
	}

	if old.State != new.State {
		update.Changes = append(update.Changes, &pb.Change{
			Type:        pb.Change_CHANGED,
			Description: fmt.Sprintf("State changed %v -> %v", old.GetState(), new.GetState()),
		})
	}

	if len(update.GetChanges()) == 0 {
		return nil
	}
	return update
}
