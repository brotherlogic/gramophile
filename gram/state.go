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
	state, err := client.GetState(ctx, &pb.GetStateRequest{})
	if err != nil {
		return err
	}

	fmt.Printf("User last updated: %v (%v folders) \n", time.Unix(state.GetLastUserRefresh(), 0), state.GetFolderCount())
	fmt.Printf("Config last udpate: %v [%v]\n", time.Unix(state.GetLastConfigUpdate(), 0), state.ConfigHash)
	fmt.Printf("Collection last synced: %v\n", time.Unix(state.GetLastCollectionSync(), 0))
	fmt.Printf("%v records in collection (%v are marked bad)\n", state.GetCollectionSize(), state.GetCollectionMisses())
	return nil
}
