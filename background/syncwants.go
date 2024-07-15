package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) CullWants(ctx context.Context, d discogs.Discogs, sid int64) error {
	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, swant := range wants {
		if swant.GetSyncId() != sid && swant.GetState() == pb.WantState_WANTED {
			swant.State = pb.WantState_RETIRED
			err = b.db.SaveWant(ctx, d.GetUserId(), swant, "Determined to be culled")
		}
	}

	return nil
}

func (b *BackgroundRunner) PullWants(ctx context.Context, d discogs.Discogs, page int32, sid int64, wc *pb.WantsConfig) (int32, error) {
	wants, pag, err := d.GetWants(ctx, page)
	log.Printf("HERE: %v", wants)

	if err != nil {
		return -1, fmt.Errorf("bad get wants: %w", err)
	}

	swants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return -1, err
	}
	for _, want := range wants {
		found := false
		for _, swant := range swants {
			if want.GetId() == swant.GetId() {
				found = true
				swant.SyncId = sid
				err := b.db.SaveWant(ctx, d.GetUserId(), swant, fmt.Sprintf("Updating on refresh (%v)", swant.GetState()))
				if err != nil {
					return -1, fmt.Errorf("error on save in pull: %w", err)
				}
				continue
			}
		}

		if !found {
			state := pb.WantState_WANTED
			if wc.GetOrigin() == pb.WantsBasis_WANTS_GRAMOPHILE {
				state = pb.WantState_RETIRED
			}
			err := b.db.SaveWant(ctx, d.GetUserId(), &pb.Want{
				Id:            want.GetId(),
				WantAddedDate: time.Now().UnixNano(),
				State:         state,
				SyncId:        sid,
			}, "Creating in sync")

			if err != nil {
				return -1, fmt.Errorf("error on new want in pull: %w", err)
			}
		}
	}

	return pag.GetPages(), nil
}
