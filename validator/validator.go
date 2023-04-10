package main

import (
	"context"
	"log"

	"google.golang.org/grpc"

	pb "github.com/brotherlogic/gramophile/proto"
)

func validateUsers(ctx context.Context) error {
	conn, err := grpc.Dial("gramophile.gramophile:8083", grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pb.NewGramophileServiceClient(conn)
	users, err := client.GetUsers(ctx, &pb.GetUsersRequest{})
	if err != nil {
		return err
	}

	return nil
}

func main() {
	log.Printf("Starting validator run")
	ctx := context.Background()

	err := validateUsers(ctx)
	if err != nil {
		log.Fatalf("Cannot validate users: %v", err)
	}
}
