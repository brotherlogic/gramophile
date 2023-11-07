package background

import (
	"context"
	"time"

	"github.com/brotherlogic/discogs"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) PullWants(ctx context.Context, d discogs.Discogs, page int32, sid int64, wc *pb.WantsConfig) (int32, error) {
	wants, pag, err := d.GetWants(ctx, page)

	if err != nil {
		return -1, err
	}

	swants, err := b.db.GetWants(ctx, d.GetUserId())
	for _, want := range wants {
		found := false
		for _, swant := range swants {
			if want.GetId() == swant.GetId() {
				if wc.GetOrigin() == pb.WantsBasis_WANTS_GRAMOPHILE {
					swant.State = pb.WantState_RETIRED
				} else {
					swant.State = pb.WantState_WANTED
				}
				found = true
				swant.SyncId = sid
				err := b.db.SaveWant(ctx, d.GetUserId(), swant)
				if err != nil {
					return -1, err
				}
				continue
			}
		}

		if !found {
			err := b.db.SaveWant(ctx, d.GetUserId(), &pb.Want{
				Id:            want.GetId(),
				WantAddedDate: time.Now().Unix(),
				State:         pb.WantState_WANTED,
				SyncId:        sid,
			})

			if err != nil {
				return -1, err
			}
		}
	}

	return pag.GetPages(), nil
}
