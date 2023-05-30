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

func ValidateConfig(ctx context.Context, fields []*pbd.Field, c *pb.GramophileConfig) error {
	cl := &cleaning{}
	err := cl.Validate(ctx, fields, c)
	return err
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

func filter(filter *pb.Filter, r *pb.Record) bool {
	for _, format := range r.GetRelease().GetFormats() {
		for _, matcher := range filter.GetFormats() {
			if matcher == format.GetName() {
				return true
			}
		}
	}

	return false
}

func Apply(c *pb.GramophileConfig, r *pb.Record) error {
	if c.GetCleaningConfig().GetCleaning() != pb.Mandate_NONE {
		if filter(c.GetCleaningConfig().GetAppliesTo(), r) {
			needsClean := false
			if c.GetCleaningConfig().GetCleaningGapInSeconds() > 0 && time.Since(time.Unix(r.GetLastCleanTime(), 0)) > time.Second*time.Duration(c.CleaningConfig.GetCleaningGapInSeconds()) {
				needsClean = true
			}

			if c.GetCleaningConfig().GetCleaningGapInPlays() > 0 && r.GetNumPlays() > c.GetCleaningConfig().GetCleaningGapInPlays() {
				needsClean = true
			}

			log.Printf("Setting for %v -> %v", r.GetRelease().GetInstanceId(), needsClean)
			setIssue(r, pb.NoncomplianceIssue_NEEDS_CLEAN, needsClean)
		}
	}

	return nil
}
