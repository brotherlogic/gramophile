package main

import (
	"context"
	"log"
	"time"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/brotherlogic/gramophile/validatorlogic"
)

func isSalesSyncActive(user *pb.StoredUser) bool {
	return validatorlogic.IsSalesSyncActive(user)
}

func validateUser(ctx context.Context, user *pb.StoredUser, client pb.GramophileServiceClient, queue pb.QueueServiceClient, d db.Database) error {
	return validatorlogic.ValidateUser(ctx, user, client, queue, d)
}

func runValidationLoop(ctx context.Context) error {
	return validatorlogic.RunValidationLoop(ctx)
}

func main() {
	log.Printf("Starting validator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	ts := time.Now()
	err := runValidationLoop(ctx)
	log.Printf("Completing validation in %v", time.Since(ts))
	if err != nil {
		log.Fatalf("Cannot run validation loop: %v", err)
	}
}
