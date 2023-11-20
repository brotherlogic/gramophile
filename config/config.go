package config

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"log"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	"google.golang.org/protobuf/proto"
)

type Validator interface {
	Validate(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error
	GetMoves(c *pb.GramophileConfig) []*pb.FolderMove
}

func ValidateConfig(ctx context.Context, user *pb.StoredUser, fields []*pbd.Field, c *pb.GramophileConfig) ([]*pbd.Folder, []*pb.FolderMove, error) {
	var moves []*pb.FolderMove
	moves = append(moves, c.GetMoves()...)

	for _, validator := range []Validator{
		&cleaning{},
		&listen{},
		&width{},
		&arrived{},
		&weight{},
		&goalFolder{},
		&sales{},
		&keep{},
		&org{},
		&sleeve{}} {
		err := validator.Validate(ctx, fields, c)
		if err != nil {
			return nil, nil, err
		}

		moves = append(moves, validator.GetMoves(c)...)
	}

	var folders []*pbd.Folder
	if c.GetCreateFolders() == pb.Create_AUTOMATIC {
		for _, move := range moves {
			if !move.GetMoveToGoalFolder() {
				folderFound := false
				for _, folder := range user.GetFolders() {
					if folder.GetName() == move.GetMoveFolder() {
						folderFound = true
					}

					if !folderFound {
						folders = append(folders, &pbd.Folder{Name: move.GetMoveFolder()})
					}
				}
			}
		}

		for _, validation := range c.GetValidations() {
			switch validation.GetValidationStrategy() {
			case pb.ValidationStrategy_LISTEN_TO_VALIDATE:
				folders = append(folders, &pbd.Folder{Name: "Listening Pile"})
			case pb.ValidationStrategy_MOVE_TO_VALIDATE:
				folders = append(folders, &pbd.Folder{Name: "Validation Pile"})
			}
		}
	}

	var rmoves []*pb.FolderMove
	if c.GetCreateMoves() == pb.Create_AUTOMATIC {
		for _, move := range moves {
			moveFound := false
			for _, umove := range user.GetMoves() {
				if move.GetName() == umove.GetName() {
					moveFound = true
				}
			}

			if !moveFound {
				move.Origin = pb.Create_AUTOMATIC
				rmoves = append(rmoves, move)
			}
		}
	}

	log.Printf("Returning folders: %v", folders)
	log.Printf("Returning moves: %v from %v", rmoves, len(moves))

	return folders, rmoves, nil
}

func Hash(c *pb.GramophileConfig) string {
	bytes, _ := proto.Marshal(c)
	hash := md5.Sum(bytes)
	return hex.EncodeToString(hash[:])
}

func setIssue(r *pb.Record, issue pb.NoncomplianceIssue, set bool) {
	found := false
	var newIssues []pb.NoncomplianceIssue
	for _, existing := range r.GetIssues() {
		if existing != issue {
			newIssues = append(newIssues, existing)
			found = true
		}
	}

	if set && !found {
		r.Issues = append(r.Issues, issue)
	}

	if !set {
		r.Issues = newIssues
	}
}

func Filter(filter *pb.Filter, r *pb.Record) bool {
	log.Printf("Folders for exclusion: %v", filter.GetExcludeFolder())
	for _, folderid := range filter.GetExcludeFolder() {
		log.Printf("Exclude %v -> %v", r.GetRelease().GetFolderId(), folderid)
		if r.GetRelease().GetFolderId() == folderid {
			return false
		}
	}
	for _, folderid := range filter.GetIncludeFolder() {
		log.Printf("Exclude %v -> %v", r, folderid)
		if r.GetRelease().GetFolderId() != folderid {
			return false
		}
	}

	for _, format := range r.GetRelease().GetFormats() {
		for _, matcher := range filter.GetFormats() {
			if matcher == format.GetName() {
				return true
			}
		}
	}

	log.Printf("HERE: %v -> %v", filter.GetFormats(), len(filter.GetFormats()))
	return len(filter.GetFormats()) == 0
}

func Apply(c *pb.GramophileConfig, r *pb.Record) error {
	if c.GetCleaningConfig().GetCleaning() != pb.Mandate_NONE {
		if Filter(c.GetCleaningConfig().GetAppliesTo(), r) {
			needsClean := false
			if c.GetCleaningConfig().GetCleaningGapInSeconds() > 0 && time.Since(time.Unix(0, r.GetLastCleanTime())) > time.Second*time.Duration(c.CleaningConfig.GetCleaningGapInSeconds()) {
				needsClean = true
			}

			if c.GetCleaningConfig().GetCleaningGapInPlays() > 0 && r.GetNumPlays() > c.GetCleaningConfig().GetCleaningGapInPlays() {
				needsClean = true
			}

			log.Printf("Setting for %v -> %v", r.GetRelease().GetInstanceId(), needsClean)
			setIssue(r, pb.NoncomplianceIssue_NEEDS_CLEAN, needsClean)
		} else {
			log.Printf("Filter %v skips %v", c.GetCleaningConfig().GetAppliesTo(), r)
			setIssue(r, pb.NoncomplianceIssue_NEEDS_CLEAN, false)
		}
	}

	return nil
}
