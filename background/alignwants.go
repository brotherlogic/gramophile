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

	cwantlist := &pb.Wantlist{Name: c.GetTransferList(), Type: pb.WantlistType_EN_MASSE}
	for _, wl := range wantlists {
		if wl.GetName() == c.GetTransferList() {
			cwantlist = wl
		}
	}

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
			cwantlist.Entries = append(cwantlist.Entries, &pb.WantlistEntry{Id: w.GetId()})
		}
	}

	return nil
}
