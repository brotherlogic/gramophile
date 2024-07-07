package background

import (
	"context"
	"fmt"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) RefreshState(ctx context.Context, iid int64, d discogs.Discogs, force bool) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	release, p, err := d.GetCollectionRelease(ctx, iid, 1)

	if err != nil {
		return err
	}

	if p.GetPages() != 1 {
		return status.Errorf(codes.Internal, "Unable to process state with > 1 pages (%v)", iid)
	}

	for _, rel := range release {
		if rel.GetInstanceId() == iid {
			//Update the elements that are pulled in the get collection retlease
			record.GetRelease().FolderId = rel.GetFolderId()

			return b.db.SaveRecord(ctx, d.GetUserId(), record)
		}
	}

	return nil
}
