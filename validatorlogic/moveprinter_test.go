package validatorlogic

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

	if lines[1] != "Gramophile Shuffle: " {
		t.Errorf("expected lines[1] to be 'Gramophile Shuffle: ', got %q", lines[1])
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

	if lines[1] != "Gramophile Move: " {
		t.Errorf("expected lines[1] to be 'Gramophile Move: ', got %q", lines[1])
	}
}

func TestRunPrintLoop_SortsByIndex(t *testing.T) {
	ctx := context.Background()

	moves := []*gpb.PrintMove{
		{
			Index:   3,
			Iid:     300,
			Record:  "Artist 3 - Album 3",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
			Printed: false,
			Origin: &gpb.Location{
				LocationName: "Shelf A",
				Shelf:        "1",
				Slot:         1,
			},
			Destination: &gpb.Location{
				LocationName: "Shelf B",
				Shelf:        "1",
				Slot:         1,
			},
		},
		{
			Index:   1,
			Iid:     100,
			Record:  "Artist 1 - Album 1",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE,
			Printed: false,
			Origin: &gpb.Location{
				LocationName: "Shelf A",
				Shelf:        "1",
				Slot:         2,
			},
			Destination: &gpb.Location{
				LocationName: "Shelf B",
				Shelf:        "1",
				Slot:         2,
			},
		},
		{
			Index:   2,
			Iid:     200,
			Record:  "Artist 2 - Album 2",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
			Printed: false,
			Origin: &gpb.Location{
				LocationName: "Shelf A",
				Shelf:        "1",
				Slot:         3,
			},
			Destination: &gpb.Location{
				LocationName: "Shelf B",
				Shelf:        "1",
				Slot:         3,
			},
		},
	}

	mockDB := &mockMoveDB{moves: moves}
	mockClient := &mockPrintQueueClient{}

	user := &gpb.StoredUser{
		User: &dpb.User{DiscogsUserId: 123},
	}

	err := runPrintLoopWithClient(ctx, mockDB, user, mockClient)
	if err != nil {
		t.Fatalf("runPrintLoopWithClient returned error: %v", err)
	}

	if len(mockClient.printedMoves) != 3 {
		t.Fatalf("expected 3 printed moves, got %d", len(mockClient.printedMoves))
	}

	expectedIndices := []string{"Index 1", "Index 2", "Index 3"}
	for i, req := range mockClient.printedMoves {
		if len(req.Lines) == 0 {
			t.Fatalf("move %d has empty lines", i)
		}
		if req.Lines[0] != expectedIndices[i] {
			t.Errorf("move %d: expected first line to be %q, got %q", i, expectedIndices[i], req.Lines[0])
		}
	}
}

func TestRunPrintLoop_SkipsAlreadyPrinted(t *testing.T) {
	ctx := context.Background()

	moves := []*gpb.PrintMove{
		{
			Index:   1,
			Iid:     100,
			Record:  "Artist 1 - Album 1",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
			Printed: true,
			Origin: &gpb.Location{
				LocationName: "Shelf A",
				Shelf:        "1",
				Slot:         1,
			},
			Destination: &gpb.Location{
				LocationName: "Shelf B",
				Shelf:        "1",
				Slot:         1,
			},
		},
		{
			Index:   2,
			Iid:     200,
			Record:  "Artist 2 - Album 2",
			Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_MOVE,
			Printed: false,
			Origin: &gpb.Location{
				LocationName: "Shelf A",
				Shelf:        "1",
				Slot:         2,
			},
			Destination: &gpb.Location{
				LocationName: "Shelf B",
				Shelf:        "1",
				Slot:         2,
			},
		},
	}

	mockDB := &mockMoveDB{moves: moves}
	mockClient := &mockPrintQueueClient{}

	user := &gpb.StoredUser{
		User: &dpb.User{DiscogsUserId: 123},
	}

	err := runPrintLoopWithClient(ctx, mockDB, user, mockClient)
	if err != nil {
		t.Fatalf("runPrintLoopWithClient returned error: %v", err)
	}

	if len(mockClient.printedMoves) != 1 {
		t.Fatalf("expected 1 printed move, got %d", len(mockClient.printedMoves))
	}

	if mockClient.printedMoves[0].Lines[0] != "Index 2" {
		t.Errorf("expected printed move to have 'Index 2', got %q", mockClient.printedMoves[0].Lines[0])
	}
}

func TestRunPrintLoop_IntegrationWithTestDB(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	tdb := db.NewTestDB(pstore)

	userId := int32(456)
	move := &gpb.PrintMove{
		Index:   1,
		Iid:     101,
		Record:  "Artist - Album",
		Type:    gpb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE,
		Printed: false,
		Origin: &gpb.Location{
			LocationName: "Shelf A",
			Shelf:        "1",
			Slot:         1,
		},
		Destination: &gpb.Location{
			LocationName: "Shelf B",
			Shelf:        "1",
			Slot:         1,
		},
	}

	err := tdb.SavePrintMove(ctx, userId, move)
	if err != nil {
		t.Fatalf("unable to save print move: %v", err)
	}

	mockClient := &mockPrintQueueClient{}
	user := &gpb.StoredUser{
		User: &dpb.User{DiscogsUserId: userId},
	}

	err = runPrintLoopWithClient(ctx, tdb, user, mockClient)
	if err != nil {
		t.Fatalf("runPrintLoopWithClient failed: %v", err)
	}

	if len(mockClient.printedMoves) != 1 {
		t.Fatalf("expected 1 move printed, got %d", len(mockClient.printedMoves))
	}

	loadedMoves, err := tdb.LoadPrintMoves(ctx, userId)
	if err != nil {
		t.Fatalf("unable to load print moves: %v", err)
	}

	if len(loadedMoves) != 1 {
		t.Fatalf("expected 1 loaded move, got %d", len(loadedMoves))
	}

	if !loadedMoves[0].Printed {
		t.Errorf("expected move to be marked printed")
	}

	if loadedMoves[0].PrintId == "" {
		t.Errorf("expected PrintId to be set")
	}
}
