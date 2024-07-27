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

	fmt.Printf("User last updated: %v (%v folders) \n", time.Unix(0, state.GetLastUserRefresh()), state.GetFolderCount())
		fmt.Printf("Sales last updated: %v\n", time.Unix(0, state.GetLastSaleRefresh()))
	fmt.Printf("Config last udpate: %v [%v]\n", time.Unix(0, state.GetLastConfigUpdate()), state.ConfigHash)
	fmt.Printf("Collection last synced: %v\n", time.Unix(0, state.GetLastCollectionSync()))
	fmt.Printf("%v records in collection (%v are marked bad)\n", state.GetCollectionSize(), state.GetCollectionMisses())
	return nil
}
