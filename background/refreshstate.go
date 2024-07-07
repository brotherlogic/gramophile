package background

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) RefreshState(ctx context.Context, iid int64, d discogs.Discogs, force bool) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		return fmt.Errorf("unable to get record from db: %w", err)
	}

	release, p, err := d.GetCollectionRelease(ctx, record.GetRelease().GetId(), 1)

	if err != nil {
		return err
	}

	if p.GetPages() != 1 {
		return status.Errorf(codes.Internal, "Unable to process state with > 1 pages (%v)", iid)
	}

	log.Printf("%v -> %v", iid, len(release))
	for _, rel := range release {
		if rel.GetInstanceId() == iid {
			//Update the elements that are pulled in the get collection retlease
			record.GetRelease().FolderId = rel.GetFolderId()

			log.Printf("Found and updated %v -> %v from %v", iid, record.GetRelease().FolderId, rel)

			return b.db.SaveRecord(ctx, d.GetUserId(), record)
		}
	}

	return nil
}
