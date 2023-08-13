package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestGoalFolderFailedNoField(t *testing.T) {
	c := &pb.GramophileConfig{GoalFolderConfig: &pb.GoalFolderConfig{Mandate: pb.Mandate_RECOMMENDED}}

	gf := &goalFolder{}
	err := gf.Validate(context.Background(), []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("Should have failed")
	}
}

func TestGoalFolderSuccess(t *testing.T) {
	c := &pb.GramophileConfig{GoalFolderConfig: &pb.GoalFolderConfig{Mandate: pb.Mandate_RECOMMENDED}}

	w := &width{}
	err := w.Validate(context.Background(), []*pbd.Field{&pbd.Field{Name: "Goal Folder", Id: 1}}, c)
	if err != nil {
		t.Errorf("Should not have failed: %v", err)
	}
}
