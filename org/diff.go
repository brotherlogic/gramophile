package org

import (
	"sort"

	pb "github.com/brotherlogic/gramophile/proto"
)

// ComputeSlotMoves compares start and end snapshots and returns moves for records
// whose physical slot (Unit) or shelf (Space) has changed. Intra-slot index shifts
// are explicitly ignored.
func ComputeSlotMoves(start, end *pb.OrganisationSnapshot) []*pb.Move {
	if start == nil || end == nil {
		return nil
	}

	endMap := make(map[int64]*pb.Placement)
	for _, p := range end.GetPlacements() {
		if p != nil {
			endMap[p.GetIid()] = p
		}
	}

	var moves []*pb.Move
	seen := make(map[int64]bool)
	for _, startP := range start.GetPlacements() {
		if startP == nil {
			continue
		}
		iid := startP.GetIid()
		if seen[iid] {
			continue
		}
		seen[iid] = true

		if endP, ok := endMap[iid]; ok && endP != nil {
			if startP.GetUnit() != endP.GetUnit() || startP.GetSpace() != endP.GetSpace() {
				moves = append(moves, &pb.Move{
					Start: startP,
					End:   endP,
				})
			}
		}
	}

	return moves
}

// OrderSlotMoves orders moves topologically such that destination slots are vacated
// before incoming records arrive (Move Y precedes Move X if Y.Start matches X.End).
// Any dependency cycles are detected and resolved by sorting cycle members by Start.Unit ascending.
func OrderSlotMoves(moves []*pb.Move) []*pb.Move {
	n := len(moves)
	if n <= 1 {
		res := make([]*pb.Move, n)
		copy(res, moves)
		return res
	}

	// Build dependency graph:
	// Move Y must precede Move X if Y.Start matches X.End.
	// Ensuring destination slots are vacated before incoming records arrive.
	adj := make([][]int, n)
	inDegree := make([]int, n)

	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			if y == x {
				continue
			}
			startY := moves[y].GetStart()
			endX := moves[x].GetEnd()
			if startY != nil && endX != nil {
				if startY.GetUnit() == endX.GetUnit() && startY.GetSpace() == endX.GetSpace() {
					adj[y] = append(adj[y], x)
					inDegree[x]++
				}
			}
		}
	}

	// Kahn's algorithm
	var queue []int
	for i := 0; i < n; i++ {
		if inDegree[i] == 0 {
			queue = append(queue, i)
		}
	}

	var ordered []*pb.Move
	visited := make([]bool, n)

	for len(queue) > 0 {
		// Sort queue by Start.Unit ascending for deterministic tie-breaking
		sort.SliceStable(queue, func(a, b int) bool {
			unitA := int32(0)
			if moves[queue[a]].GetStart() != nil {
				unitA = moves[queue[a]].GetStart().GetUnit()
			}
			unitB := int32(0)
			if moves[queue[b]].GetStart() != nil {
				unitB = moves[queue[b]].GetStart().GetUnit()
			}
			return unitA < unitB
		})

		curr := queue[0]
		queue = queue[1:]

		ordered = append(ordered, moves[curr])
		visited[curr] = true

		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	// Detect dependency cycles: any node not visited is part of or blocked by a cycle
	if len(ordered) < n {
		var cycleMoves []*pb.Move
		for i := 0; i < n; i++ {
			if !visited[i] {
				cycleMoves = append(cycleMoves, moves[i])
			}
		}

		// Resolve cycle members by sorting them by Start.Unit ascending
		sort.SliceStable(cycleMoves, func(a, b int) bool {
			unitA := int32(0)
			if cycleMoves[a].GetStart() != nil {
				unitA = cycleMoves[a].GetStart().GetUnit()
			}
			unitB := int32(0)
			if cycleMoves[b].GetStart() != nil {
				unitB = cycleMoves[b].GetStart().GetUnit()
			}
			return unitA < unitB
		})

		ordered = append(ordered, cycleMoves...)
	}

	return ordered
}
