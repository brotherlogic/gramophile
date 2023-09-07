package background

import (
	"context"
	"fmt"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) AddFolder(ctx context.Context, folderName string, d discogs.Discogs, u *pb.StoredUser) error {
	folder, err := d.CreateFolder(ctx, folderName)
	if err != nil {
		return fmt.Errorf("unable to create folder: %w", err)
	}

	u.Folders = append(u.Folders, folder)
	return b.db.SaveUser(ctx, u)
}
