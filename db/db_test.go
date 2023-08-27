package db

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestUpdateDiff(t *testing.T) {
	tests := []struct {
		update  *pb.RecordUpdate
		diffstr string
	}{
		{
			update: &pb.RecordUpdate{
				Before: &pb.Record{
					GoalFolder: "",
				},
				After: &pb.Record{
					GoalFolder: "12 Inches",
				},
			},
			diffstr: "Goal Folder was set to 12 Inches",
		},
	}

	for _, tc := range tests {
		rd := resolveDiff(tc.update)
		found := false
		for _, rdd := range rd {
			if rdd == tc.diffstr {
				found = true
			}
		}
		if !found {
			t.Errorf("bad diff: expected %v, got %v", tc.diffstr, rd)
		}

	}
}
