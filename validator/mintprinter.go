package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/brotherlogic/gramophile/db"

	gpb "github.com/brotherlogic/gramophile/proto"
	pqpb "github.com/brotherlogic/printqueue/proto"

	printqueueclient "github.com/brotherlogic/printqueue/client"
)

func runMintPrinter(ctx context.Context, user *gpb.StoredUser) error {
	db := db.NewDatabase(ctx)

	log.Printf("Mint Up %v vs %v", time.Unix(0, user.GetConfig().GetMintUpConfig().GetLastMintUpDelivery()), user.GetConfig().GetMintUpConfig().GetPeriodInSeconds())
	// Don't send mint ups if the user doesn't want them.
	if time.Since(time.Unix(0, user.GetConfig().GetMintUpConfig().GetLastMintUpDelivery())).Seconds() < float64(user.GetConfig().GetMintUpConfig().GetPeriodInSeconds()) ||
		user.GetConfig().GetMintUpConfig().GetPeriodInSeconds() == 0 {
		return nil
	}

	records, err := db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return err
	}

	rand.Shuffle(len(records), func(i, j int) {
		records[i], records[j] = records[j], records[i]
	})

	for _, r := range records {
		rec, err := db.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return err
		}

		if rec.GetKeepStatus() == gpb.KeepStatus_MINT_UP_KEEP {
			pclient, err := printqueueclient.NewPrintQueueClient(ctx)
			if err != nil {
				return err
			}

			_, err = pclient.Print(ctx, &pqpb.PrintRequest{
				Origin:      "mint_printer",
				Destination: pqpb.Destination_DESTINATION_RECEIPT,
				Urgency:     pqpb.Urgency_URGENCY_REGULAR,
				Fanout:      pqpb.Fanout_FANOUT_ONE,
				Lines:       []string{fmt.Sprintf("Mintup: %v - %v", rec.GetRelease().GetArtists()[0].GetName(), rec.GetRelease().GetTitle())},
			})
			if err != nil {
				return err
			}

			user.GetConfig().GetMintUpConfig().LastMintUpDelivery = time.Now().UnixNano()
			return db.SaveUser(ctx, user)
		}

	}

	return nil
}
