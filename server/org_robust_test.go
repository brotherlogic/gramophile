package server

import (
	"math/rand"
	"testing"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSnapshotDiffRobust_NoOp(t *testing.T) {
	snap := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Index: 1, Space: "S1", Unit: 1},
			{Iid: 2, Index: 2, Space: "S1", Unit: 1},
		},
	}

	moves := getSnapshotDiff(snap, snap)
	if len(moves) != 0 {
		t.Errorf("Expected 0 moves for identical snapshots, got %d", len(moves))
	}
}

func TestSnapshotDiffRobust_SpaceUnitTransition(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Index: 1, Space: "S1", Unit: 1},
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Index: 1, Space: "S2", Unit: 2},
		},
	}

	moves := getSnapshotDiff(start, end)
	if len(moves) != 1 {
		t.Errorf("Expected 1 move, got %d", len(moves))
	} else {
		if moves[0].Start.Space != "S1" || moves[0].End.Space != "S2" {
			t.Errorf("Unexpected move: %v", moves[0])
		}
	}
}

func TestSnapshotDiffRobust_ReverseShuffle(t *testing.T) {
	numRecords := 10
	startPlacements := make([]*pb.Placement, numRecords)
	endPlacements := make([]*pb.Placement, numRecords)

	for i := 0; i < numRecords; i++ {
		startPlacements[i] = &pb.Placement{Iid: int64(i + 1), Index: int32(i + 1), Space: "S1", Unit: 1}
		endPlacements[i] = &pb.Placement{Iid: int64(numRecords - i), Index: int32(i + 1), Space: "S1", Unit: 1}
	}

	start := &pb.OrganisationSnapshot{Placements: startPlacements}
	end := &pb.OrganisationSnapshot{Placements: endPlacements}

	moves := getSnapshotDiff(start, end)
	
	nsnap := applyMoves(start, moves)
	if tgetHash(end.GetPlacements()) != tgetHash(nsnap.GetPlacements()) {
		t.Errorf("Reverse shuffle moves failed to result in correct end state")
	}
}

func TestSnapshotDiffRobust_LargeScaleShuffle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large scale shuffle in short mode")
	}

	numRecords := 1000
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	startPlacements := make([]*pb.Placement, numRecords)
	for i := 0; i < numRecords; i++ {
		startPlacements[i] = &pb.Placement{Iid: int64(i + 1), Index: int32(i + 1), Space: "S1", Unit: 1}
	}
	start := &pb.OrganisationSnapshot{Placements: startPlacements}

	// Create a shuffled end state
	p := r.Perm(numRecords)
	endPlacements := make([]*pb.Placement, numRecords)
	for i := 0; i < numRecords; i++ {
		endPlacements[i] = &pb.Placement{Iid: int64(p[i] + 1), Index: int32(i + 1), Space: "S1", Unit: 1}
	}
	end := &pb.OrganisationSnapshot{Placements: endPlacements}

	t1 := time.Now()
	moves := getSnapshotDiff(start, end)
	duration := time.Since(t1)

	if duration > time.Second {
		t.Errorf("getSnapshotDiff took too long: %v", duration)
	}

	nsnap := applyMoves(start, moves)
	if tgetHash(end.GetPlacements()) != tgetHash(nsnap.GetPlacements()) {
		t.Errorf("Large scale shuffle moves failed")
	}
}

func TestSnapshotDiffRobust_PartialOverlap(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Index: 1, Space: "S1", Unit: 1},
			{Iid: 2, Index: 2, Space: "S1", Unit: 1},
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 2, Index: 1, Space: "S1", Unit: 1},
			{Iid: 3, Index: 2, Space: "S1", Unit: 1},
		},
	}

	moves := getSnapshotDiff(start, end)
	
	foundAddition := false
	foundDeletion := false
	foundMove := false

	for _, m := range moves {
		if m.GetStart() == nil && m.GetEnd().GetIid() == 3 {
			foundAddition = true
		}
		if m.GetEnd() == nil && m.GetStart().GetIid() == 1 {
			foundDeletion = true
		}
		if m.GetStart() != nil && m.GetEnd() != nil && m.GetStart().GetIid() == 2 {
			foundMove = true
		}
	}

	if !foundAddition {
		t.Errorf("Did not find expected addition move for Iid 3")
	}
	if !foundDeletion {
		t.Errorf("Did not find expected deletion move for Iid 1")
	}
	if !foundMove {
		t.Errorf("Did not find expected move for Iid 2")
	}

	nsnap := applyMoves(start, moves)
	if tgetHash(end.GetPlacements()) != tgetHash(nsnap.GetPlacements()) {
		t.Errorf("Partial overlap moves failed to result in correct end state")
	}
}
