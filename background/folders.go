package background

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
)

type addFolderUpdateHandler struct {
	b *BackgroundRunner
}

func (h *addFolderUpdateHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.AddFolder(ctx, entry.GetAddFolderUpdate().GetFolderName(), d, u)
}

func (h *addFolderUpdateHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *addFolderUpdateHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) AddFolder(ctx context.Context, folderName string, d discogs.Discogs, u *pb.StoredUser) error {
	log.Printf("Creating folder: %v", folderName)
	folder, err := d.CreateFolder(ctx, folderName)
	if err != nil {
		return fmt.Errorf("unable to create folder: %w", err)
	}

	u.Folders = append(u.Folders, folder)
	return b.db.SaveUser(ctx, u)
}
