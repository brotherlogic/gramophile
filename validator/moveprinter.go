package main

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/brotherlogic/gramophile/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	gpb "github.com/brotherlogic/gramophile/proto"
	pqpb "github.com/brotherlogic/printqueue/proto"

	printqueueclient "github.com/brotherlogic/printqueue/client"
)

var (
	printQueueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_print_queue_len",
		Help: "The length of the working queue I think yes",
	})
)

func buildRepresentation(move *gpb.PrintMove) []string {
	var lines []string

	lines = append(lines, fmt.Sprintf("Index %v", move.GetIndex()))
	if move.GetType() == gpb.PrintMoveType_PRINT_MOVE_TYPE_SHUFFLE {
		lines = append(lines, "Gramophile Shuffle: ")
	} else {
		lines = append(lines, "Gramophile Move: ")
	}
	lines = append(lines, fmt.Sprintf("%v", move.GetRecord()))

	lines = append(lines, fmt.Sprintf("%v [%v-%v]", move.GetOrigin().GetLocationName(), move.GetOrigin().GetShelf(), move.GetOrigin().GetSlot()))
	for _, c := range move.GetOrigin().GetBefore() {
		lines = append(lines, fmt.Sprintf("%v", c.GetRecord()))
	}
	lines = append(lines, fmt.Sprintf("%v", move.GetRecord()))
	for _, c := range move.GetOrigin().GetAfter() {
		lines = append(lines, fmt.Sprintf("%v", c.GetRecord()))
	}

	lines = append(lines, "")

	lines = append(lines, fmt.Sprintf("%v [%v-%v]", move.GetDestination().GetLocationName(), move.GetDestination().GetShelf(), move.GetDestination().GetSlot()))
	for _, c := range move.GetDestination().GetBefore() {
		lines = append(lines, fmt.Sprintf("%v", c.GetRecord()))
	}
	lines = append(lines, fmt.Sprintf("%v", move.GetRecord()))
	for _, c := range move.GetDestination().GetAfter() {
		lines = append(lines, fmt.Sprintf("%v", c.GetRecord()))
	}

	return lines
}

type printQueueClient interface {
	Print(ctx context.Context, req *pqpb.PrintRequest) (*pqpb.PrintResponse, error)
}

func runPrintLoop(ctx context.Context, d db.Database, user *gpb.StoredUser) error {
	pClient, err := printqueueclient.NewPrintQueueClient(ctx)
	if err != nil {
		return err
	}
	return runPrintLoopWithClient(ctx, d, user, pClient)
}

func runPrintLoopWithClient(ctx context.Context, d db.Database, user *gpb.StoredUser, pClient printQueueClient) error {
	if d == nil {
		d = db.NewDatabase(ctx)
	}

	moves, err := d.LoadPrintMoves(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return err
	}

	var unprinted []*gpb.PrintMove
	for _, move := range moves {
		if !move.Printed {
			unprinted = append(unprinted, move)
		}
	}

	log.Printf("Found %v moves (%v unprinted)", len(moves), len(unprinted))

	printQueueLen.Set(float64(len(moves)))

	sort.Slice(unprinted, func(i, j int) bool {
		return unprinted[i].GetIndex() < unprinted[j].GetIndex()
	})

	for _, move := range unprinted {
		lines := buildRepresentation(move)

		resp, err := pClient.Print(ctx, &pqpb.PrintRequest{
			Lines:       lines,
			Origin:      "gram-move-loop",
			Urgency:     pqpb.Urgency_URGENCY_REGULAR,
			Destination: pqpb.Destination_DESTINATION_RECEIPT,
			Fanout:      pqpb.Fanout_FANOUT_ONE,
		})

		if err == nil {

			move.Printed = true
			move.PrintId = resp.GetId()
			err = d.SavePrintMove(ctx, user.GetUser().GetDiscogsUserId(), move)
			log.Printf("Deleted print move for %v -> %v (%v)", move.GetIid(), err, move)
		} else {
			log.Printf("Failed to move %v %v", resp, err)
		}
	}

	return nil
}
