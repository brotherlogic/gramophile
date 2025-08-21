package db

import (
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func buildWantUpdates(old, new *pb.Want, up *pb.WantUpdate, reason string) *pb.WantUpdate {
	update := &pb.Update{Date: time.Now().UnixNano()}

	if old == nil {
		update.Changes = append(update.Changes, &pb.Change{
			Type:        pb.Change_ADDED,
			Description: fmt.Sprintf("Want created: %v", reason),
		})
		up.Updates = append(up.Updates, update)
		return up
	}

	if old.State != new.State {
		update.Changes = append(update.Changes, &pb.Change{
			Type:        pb.Change_CHANGED,
			Description: fmt.Sprintf("State changed %v -> %v (%v)", old.GetState(), new.GetState(), reason),
		})
	}

	if old.State != new.IntendedState {
		update.Changes = append(update.Changes, &pb.Change{
			Type:        pb.Change_CHANGED,
			Description: fmt.Sprintf("State changed: %v -> %v (%v)", old.GetState(), new.GetIntendedState(), reason),
		})
	}

	if len(update.GetChanges()) == 0 {
		return nil
	}

	up.Updates = append(up.Updates, update)

	return up
}
