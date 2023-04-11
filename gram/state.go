package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGetSate() *CLIModule {
	return &CLIModule{
		Command: "state",
		Help:    "Get the current state of the system",
		Execute: executeGetState,
	}
}

func executeGetState(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	user, err := client.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		return err
	}

	fmt.Printf("User last updated: %v\n", time.Unix(user.GetLastUserRefresh(), 0))
	return nil
}
