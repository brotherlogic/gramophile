package background

import (
	"context"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

// Ensures everything is in a wantlist - only used when ORIGIN_GRAMOPHILE is set for wants
func (b *BackgroundRunner) AlignWants(ctx context.Context, di discogs.Discogs, c *pb.WantsConfig) error {

	wants, err := b.db.GetWants(ctx, di.GetUserId())
	if err != nil {
		return err
	}

	wantlists, err := b.db.GetWantlists(ctx, di.GetUserId())
	if err != nil {
		return err
	}

	cwantlist := &pb.Wantlist{Name: c.GetTransferList(), Type: pb.WantlistType_ONE_BY_ONE}
	for _, wl := range wantlists {
		if wl.GetName() == c.GetTransferList() {
			cwantlist = wl
		}
	}

	updated := false
	for _, w := range wants {
		found := false
		for _, wl := range wantlists {
			for _, entry := range wl.GetEntries() {
				if entry.GetId() == w.GetId() {
					found = true
					break
				}
			}
		}

		if !found {
			if c.GetExisting() == pb.WantsExisting_EXISTING_DROP {
				w.State = pb.WantState_RETIRED
				b.db.SaveWant(ctx, di.GetUserId(), w, "Config is set to EXISTING_DROP")
			} else {
				updated = true
				cwantlist.Entries = append(cwantlist.Entries, &pb.WantlistEntry{Id: w.GetId()})
			}
		}
	}

	if updated {
		return b.db.SaveWantlist(ctx, di.GetUserId(), cwantlist)
	}

	return nil
}
