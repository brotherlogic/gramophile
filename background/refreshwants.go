package background

import (
	"context"
	"fmt"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) RefereshWants(ctx context.Context, d discogs.Discogs) error {
	// Look for any wants that have been purchased
	recs, err := b.db.LoadAllRecords(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all records: %w", err)
	}

	wants, err := b.db.GetWants(ctx, d.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to load all wants: %w", err)
	}

	for _, want := range wants {
		for _, rec := range recs {
			if want.GetId() == rec.GetRelease().GetId() {
				want.State = pb.WantState_PURCHASED
				err := b.db.SaveWant(ctx, d.GetUserId(), want)
				if err != nil {
					return fmt.Errorf("unable to save want: %w", err)
				}
				continue
			}
		}
	}

	return nil
}
