package background

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"time"
)

type refreshStateHandler struct {
	b *BackgroundRunner
}

func (h *refreshStateHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.ProcessRefreshState(ctx, d, entry, enqueue)
}

func (h *refreshStateHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *refreshStateHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) ProcessRefreshState(ctx context.Context, d discogs.Discogs, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	err := b.RefreshState(ctx, entry.GetRefreshState().GetIid(), d, entry.GetRefreshState().GetForce())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			_, err := enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Auth:      entry.GetAuth(),
					Force:     true,
					RunDate:   time.Now().UnixNano(),
					Intention: fmt.Sprintf("Refreshing collection from release state %v", entry.GetRefreshState().GetIid()),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
					},
				},
			})
			if err != nil && status.Code(err) != codes.ResourceExhausted {
				return err
			}
		}
	}
	return err
}

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

			if record.GetRelease().DateAdded == 0 {
				record.GetRelease().DateAdded = rel.GetDateAdded()
			}

			log.Printf("Found and updated %v -> %v from %v", iid, record.GetRelease().FolderId, rel)

			return b.db.SaveRecord(ctx, d.GetUserId(), record, &db.SaveOptions{})
		}
	}

	return nil
}
