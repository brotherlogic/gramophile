package validator

import (
	"context"
	"fmt"

	"github.com/brotherlogic/gramophile/db"

	gpb "github.com/brotherlogic/gramophile/proto"
	pqpb "github.com/brotherlogic/printqueue/proto"

	printqueueclient "github.com/brotherlogic/printqueue/client"
)

func buildRepresentation(move *gpb.PrintMove) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("%v", move.GetRecord()))

	lines = append(lines, fmt.Sprintf("%v", move.GetBefore().GetLocationName()))
	for _, c := range move.GetBefore().GetContext() {
		lines = append(lines, fmt.Sprintf("%v", c.GetIid()))
	}
	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%v", move.GetAfter().GetLocationName()))
	for _, c := range move.GetAfter().GetContext() {
		lines = append(lines, fmt.Sprintf("%v", c.GetIid()))
	}

	return lines
}

func runPrintLoop(ctx context.Context, uid string) error {
	db := db.NewDatabase(ctx)

	user, err := db.GetUser(ctx, uid)

	moves, err := db.LoadPrintMoves(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return err
	}

	pClient, err := printqueueclient.NewPrintQueueClient(ctx)
	if err != nil {
		return err
	}
	for _, move := range moves {
		lines := buildRepresentation(move)

		_, err = pClient.Print(ctx, &pqpb.PrintRequest{
			Lines:       lines,
			Origin:      "gram-move-loop",
			Urgency:     pqpb.Urgency_URGENCY_REGULAR,
			Destination: pqpb.Destination_DESTINATION_RECEIPT,
			Fanout:      pqpb.Fanout_FANOUT_ONE,
		})

		if err == nil {
			db.DeletePrintMove(ctx, user.GetUser().GetDiscogsUserId(), move.GetInstanceId())
		}
	}
}
