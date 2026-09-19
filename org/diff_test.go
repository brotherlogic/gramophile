package org

import (
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSlotDiff_IntraSlotIndexShiftIgnored(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Space: "S1", Unit: 1, Index: 1},
			{Iid: 2, Space: "S1", Unit: 1, Index: 2},
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Space: "S1", Unit: 1, Index: 2},
			{Iid: 2, Space: "S1", Unit: 1, Index: 1},
		},
	}

	moves := ComputeSlotMoves(start, end)
	if len(moves) != 0 {
		t.Fatalf("Expected 0 moves for intra-slot index shift, got %d: %v", len(moves), moves)
	}
}

func TestSlotDiff_SlotCascadeOrdering(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 101, Space: "S1", Unit: 1, Index: 1},
			{Iid: 102, Space: "S1", Unit: 2, Index: 1},
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 101, Space: "S1", Unit: 2, Index: 1},
			{Iid: 102, Space: "S1", Unit: 3, Index: 1},
		},
	}

	moves := ComputeSlotMoves(start, end)
	if len(moves) != 2 {
		t.Fatalf("Expected 2 moves, got %d", len(moves))
	}

	ordered := OrderSlotMoves(moves)
	if len(ordered) != 2 {
		t.Fatalf("Expected 2 ordered moves, got %d", len(ordered))
	}

	// Move 2->3 (vacating Unit 2) must precede Move 1->2 (entering Unit 2)
	if ordered[0].GetStart().GetUnit() != 2 || ordered[0].GetEnd().GetUnit() != 3 {
		t.Errorf("Expected first move to be 2->3, got %d->%d",
			ordered[0].GetStart().GetUnit(), ordered[0].GetEnd().GetUnit())
	}
	if ordered[1].GetStart().GetUnit() != 1 || ordered[1].GetEnd().GetUnit() != 2 {
		t.Errorf("Expected second move to be 1->2, got %d->%d",
			ordered[1].GetStart().GetUnit(), ordered[1].GetEnd().GetUnit())
	}
}

func TestSlotDiff_CycleFallback(t *testing.T) {
	// Circular dependency: Slot 2 -> Slot 1 and Slot 1 -> Slot 2
	m1 := &pb.Move{
		Start: &pb.Placement{Iid: 201, Space: "S1", Unit: 2},
		End:   &pb.Placement{Iid: 201, Space: "S1", Unit: 1},
	}
	m2 := &pb.Move{
		Start: &pb.Placement{Iid: 202, Space: "S1", Unit: 1},
		End:   &pb.Placement{Iid: 202, Space: "S1", Unit: 2},
	}

	// Input in arbitrary order (m1 then m2)
	ordered := OrderSlotMoves([]*pb.Move{m1, m2})
	if len(ordered) != 2 {
		t.Fatalf("Expected 2 ordered moves, got %d", len(ordered))
	}

	// Cycle members should be resolved by sorting them by Start.Unit ascending (Unit 1 before Unit 2)
	if ordered[0].GetStart().GetUnit() != 1 || ordered[1].GetStart().GetUnit() != 2 {
		t.Errorf("Expected cycle resolved by Start.Unit ascending (1, 2), got %d, %d",
			ordered[0].GetStart().GetUnit(), ordered[1].GetStart().GetUnit())
	}
}

func TestSlotDiff_NilSnapshots(t *testing.T) {
	if moves := ComputeSlotMoves(nil, nil); moves != nil {
		t.Errorf("Expected nil moves for nil snapshots, got %v", moves)
	}
	start := &pb.OrganisationSnapshot{}
	if moves := ComputeSlotMoves(start, nil); moves != nil {
		t.Errorf("Expected nil moves for nil end snapshot, got %v", moves)
	}
	if moves := ComputeSlotMoves(nil, start); moves != nil {
		t.Errorf("Expected nil moves for nil start snapshot, got %v", moves)
	}
}

func TestSlotDiff_SpaceChange(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Space: "ShelfA", Unit: 1, Index: 1},
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Space: "ShelfB", Unit: 1, Index: 1},
		},
	}

	moves := ComputeSlotMoves(start, end)
	if len(moves) != 1 {
		t.Fatalf("Expected 1 move for space change, got %d", len(moves))
	}
	if moves[0].GetStart().GetSpace() != "ShelfA" || moves[0].GetEnd().GetSpace() != "ShelfB" {
		t.Errorf("Unexpected move: %v", moves[0])
	}
}

func TestSlotDiff_IgnoresAdditionsAndDeletions(t *testing.T) {
	start := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 1, Space: "S1", Unit: 1, Index: 1}, // deleted in end
			{Iid: 2, Space: "S1", Unit: 2, Index: 1}, // stays in unit 2
		},
	}
	end := &pb.OrganisationSnapshot{
		Placements: []*pb.Placement{
			{Iid: 2, Space: "S1", Unit: 2, Index: 2}, // intra-slot index change
			{Iid: 3, Space: "S1", Unit: 3, Index: 1}, // added in end
		},
	}

	moves := ComputeSlotMoves(start, end)
	if len(moves) != 0 {
		t.Fatalf("Expected 0 slot moves, got %d: %v", len(moves), moves)
	}
}

func TestSlotDiff_ThreeWayCycle(t *testing.T) {
	// 3-way cycle: 3 -> 1, 1 -> 2, 2 -> 3
	m3 := &pb.Move{
		Start: &pb.Placement{Iid: 301, Space: "S1", Unit: 3},
		End:   &pb.Placement{Iid: 301, Space: "S1", Unit: 1},
	}
	m1 := &pb.Move{
		Start: &pb.Placement{Iid: 302, Space: "S1", Unit: 1},
		End:   &pb.Placement{Iid: 302, Space: "S1", Unit: 2},
	}
	m2 := &pb.Move{
		Start: &pb.Placement{Iid: 303, Space: "S1", Unit: 2},
		End:   &pb.Placement{Iid: 303, Space: "S1", Unit: 3},
	}

	ordered := OrderSlotMoves([]*pb.Move{m3, m2, m1})
	if len(ordered) != 3 {
		t.Fatalf("Expected 3 ordered moves, got %d", len(ordered))
	}

	// Should be sorted by Start.Unit ascending: 1, 2, 3
	if ordered[0].GetStart().GetUnit() != 1 || ordered[1].GetStart().GetUnit() != 2 || ordered[2].GetStart().GetUnit() != 3 {
		t.Errorf("Expected 3-way cycle resolved by Start.Unit ascending (1, 2, 3), got %d, %d, %d",
			ordered[0].GetStart().GetUnit(), ordered[1].GetStart().GetUnit(), ordered[2].GetStart().GetUnit())
	}
}

func TestSlotDiff_MixedAcyclicAndCycle(t *testing.T) {
	// Independent move: 10 -> 20
	mIndep := &pb.Move{
		Start: &pb.Placement{Iid: 999, Space: "S1", Unit: 10},
		End:   &pb.Placement{Iid: 999, Space: "S1", Unit: 20},
	}
	// Cycle: 2 -> 1, 1 -> 2
	mCycle1 := &pb.Move{
		Start: &pb.Placement{Iid: 201, Space: "S1", Unit: 2},
		End:   &pb.Placement{Iid: 201, Space: "S1", Unit: 1},
	}
	mCycle2 := &pb.Move{
		Start: &pb.Placement{Iid: 202, Space: "S1", Unit: 1},
		End:   &pb.Placement{Iid: 202, Space: "S1", Unit: 2},
	}

	ordered := OrderSlotMoves([]*pb.Move{mCycle1, mIndep, mCycle2})
	if len(ordered) != 3 {
		t.Fatalf("Expected 3 ordered moves, got %d", len(ordered))
	}

	// The independent move should be resolved first by Kahn's algorithm
	if ordered[0].GetStart().GetUnit() != 10 {
		t.Errorf("Expected independent move 10->20 first, got %d->%d",
			ordered[0].GetStart().GetUnit(), ordered[0].GetEnd().GetUnit())
	}
	// The cycle moves should follow, sorted by Start.Unit ascending (1 then 2)
	if ordered[1].GetStart().GetUnit() != 1 || ordered[2].GetStart().GetUnit() != 2 {
		t.Errorf("Expected cycle moves sorted by Start.Unit ascending (1, 2), got %d, %d",
			ordered[1].GetStart().GetUnit(), ordered[2].GetStart().GetUnit())
	}
}
