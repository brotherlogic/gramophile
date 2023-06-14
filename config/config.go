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
}

func ValidateConfig(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	for _, validator := range []Validator{&cleaning{}, &listen{}} {
		err := validator.Validate(ctx, fields, c)
		if err != nil {
			return err
		}
	}

	return nil
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
	return len(filter.GetFormats()) == 0
}

func Apply(c *pb.GramophileConfig, r *pb.Record) error {
	if c.GetCleaningConfig().GetCleaning() != pb.Mandate_NONE {
		if Filter(c.GetCleaningConfig().GetAppliesTo(), r) {
			needsClean := false
			if c.GetCleaningConfig().GetCleaningGapInSeconds() > 0 && time.Since(time.Unix(r.GetLastCleanTime(), 0)) > time.Second*time.Duration(c.CleaningConfig.GetCleaningGapInSeconds()) {
				needsClean = true
			}

			if c.GetCleaningConfig().GetCleaningGapInPlays() > 0 && r.GetNumPlays() > c.GetCleaningConfig().GetCleaningGapInPlays() {
				needsClean = true
			}

			setIssue(r, pb.NoncomplianceIssue_NEEDS_CLEAN, needsClean)
		} else {
			log.Printf("Filter %v skips %v", c.GetCleaningConfig().GetAppliesTo(), r)
			setIssue(r, pb.NoncomplianceIssue_NEEDS_CLEAN, false)
		}
	}

	return nil
}
