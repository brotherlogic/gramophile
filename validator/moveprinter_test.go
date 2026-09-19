package main

import (
	"context"
	"fmt"
	"testing"

	dpb "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	gpb "github.com/brotherlogic/gramophile/proto"
	pqpb "github.com/brotherlogic/printqueue/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

type mockPrintQueueClient struct {
	printedMoves []*pqpb.PrintRequest
}

func (m *mockPrintQueueClient) Print(ctx context.Context, req *pqpb.PrintRequest) (*pqpb.PrintResponse, error) {
	m.printedMoves = append(m.printedMoves, req)
	return &pqpb.PrintResponse{Id: fmt.Sprintf("print-%d", len(m.printedMoves))}, nil
}

type mockMoveDB struct {
	db.Database
	moves []*gpb.PrintMove
	saved []*gpb.PrintMove
}

func (m *mockMoveDB) LoadPrintMoves(ctx context.Context, userId int32) ([]*gpb.PrintMove, error) {
	return m.moves, nil
}

func (m *mockMoveDB) SavePrintMove(ctx context.Context, userId int32, move *gpb.PrintMove) error {
	m.saved = append(m.saved, move)
	return nil
}

func TestBuildRepresentation_Shuffle(t *testing.T) {
	move := &gpb.PrintMove{
		Index:  1,
		Record: "Test Artist - Test Album",
		Type:   gpb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE,
		Origin: &gpb.Location{
			LocationName: "Shelf A",
			Shelf:        "1",
			Slot:         2,
		},
		Destination: &gpb.Location{
			LocationName: "Shelf B",
			Shelf:        "1",
			Slot:         3,
		},
	}

	lines := buildRepresentation(move)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %v", lines)
	}

	expectedHeader := "Gramophile Shuffle: "
	if lines[1] != expectedHeader {
		t.Errorf("expected lines[1] to be %q, got %q", expectedHeader, lines[1])
	}
}

func TestBuildRepresentation_Move(t *testing.T) {
	move := &gpb.PrintMove{
		Index:  1,
		Record: "Test Artist - Test Album",
		Type:   gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
		Origin: &gpb.Location{
			LocationName: "Shelf A",
			Shelf:        "1",
			Slot:         2,
		},
		Destination: &gpb.Location{
			LocationName: "Shelf B",
			Shelf:        "1",
			Slot:         3,
		},
	}

	lines := buildRepresentation(move)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %v", lines)
	}

	expectedHeader := "Gramophile Move: "
	if lines[1] != expectedHeader {
		t.Errorf("expected lines[1] to be %q, got %q", expectedHeader, lines[1])
	}
}

func TestBuildRepresentation_UnspecifiedFallback(t *testing.T) {
	move := &gpb.PrintMove{
		Index:  1,
		Record: "Test Artist - Test Album",
		Type:   gpb.PrintMoveType_PRINT_MOVE_TYPE_UNSPECIFIED,
		Origin: &gpb.Location{
			LocationName: "Shelf A",
			Shelf:        "1",
			Slot:         2,
		},
		Destination: &gpb.Location{
			LocationName: "Shelf B",
			Shelf:        "1",
			Slot:         3,
		},
	}

	lines := buildRepresentation(move)
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 lines, got %v", lines)
	}

	expectedHeader := "Gramophile Move: "
	if lines[1] != expectedHeader {
		t.Errorf("expected lines[1] to be %q, got %q", expectedHeader, lines[1])
	}
}

func TestRunPrintLoop_IndexSortOrder(t *testing.T) {
	ctx := context.Background()
	user := &gpb.StoredUser{
		User: &dpb.User{DiscogsUserId: 1234},
	}

	// Deliberately unsorted moves returned by DB
	moves := []*gpb.PrintMove{
		{
			Iid:     101,
			Index:   5,
			Printed: false,
			Record:  "Album 5",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE,
		},
		{
			Iid:     102,
			Index:   1,
			Printed: false,
			Record:  "Album 1",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
		},
		{
			Iid:     103,
			Index:   3,
			Printed: false,
			Record:  "Album 3",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
		},
		{
			Iid:     104,
			Index:   2,
			Printed: true, // Already printed, must be skipped
			Record:  "Album 2",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
		},
	}

	tdb := &mockMoveDB{
		moves: moves,
	}

	mockClient := &mockPrintQueueClient{}
	err := runPrintLoopWithClient(ctx, tdb, user, mockClient)
	if err != nil {
		t.Fatalf("runPrintLoopWithClient failed: %v", err)
	}

	if len(mockClient.printedMoves) != 3 {
		t.Fatalf("expected 3 printed moves, got %d", len(mockClient.printedMoves))
	}

	// Verify the order of prints strictly follows ascending Index order: 1, 3, 5
	expectedOrder := []string{"Index 1", "Index 3", "Index 5"}
	for i, exp := range expectedOrder {
		if len(mockClient.printedMoves[i].GetLines()) == 0 || mockClient.printedMoves[i].GetLines()[0] != exp {
			t.Errorf("move %d expected first line %q, got %v", i, exp, mockClient.printedMoves[i].GetLines())
		}
	}

	if len(tdb.saved) != 3 {
		t.Fatalf("expected 3 saved moves, got %d", len(tdb.saved))
	}
	for _, sm := range tdb.saved {
		if !sm.GetPrinted() {
			t.Errorf("move %v should be marked printed", sm.GetIid())
		}
	}
}

func TestRunPrintLoop_IntegrationWithTestDB(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	user := &gpb.StoredUser{
		User: &dpb.User{DiscogsUserId: 5678},
	}

	move := &gpb.PrintMove{
		Iid:     201,
		Index:   10,
		Printed: false,
		Record:  "Integration Album",
		Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
	}
	if err := tdb.SavePrintMove(ctx, user.GetUser().GetDiscogsUserId(), move); err != nil {
		t.Fatalf("SavePrintMove failed: %v", err)
	}

	mockClient := &mockPrintQueueClient{}
	err := runPrintLoopWithClient(ctx, tdb, user, mockClient)
	if err != nil {
		t.Fatalf("runPrintLoopWithClient failed: %v", err)
	}

	if len(mockClient.printedMoves) != 1 {
		t.Fatalf("expected 1 printed move, got %d", len(mockClient.printedMoves))
	}

	storedMoves, err := tdb.LoadPrintMoves(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		t.Fatalf("LoadPrintMoves failed: %v", err)
	}
	if len(storedMoves) != 1 || !storedMoves[0].GetPrinted() {
		t.Errorf("expected 1 printed move in DB, got %v", storedMoves)
	}
}
