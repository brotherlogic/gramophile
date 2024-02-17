package main

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func GetWant() *CLIModule {
	return &CLIModule{
		Command: "want",
		Help:    "want",
		Execute: executeWant,
	}
}

func executeWant(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	client := pb.NewGramophileEServiceClient(conn)

	// This is just a list
	if len(args) == 0 {
		wants, err := client.GetWants(ctx, &pb.GetWantsRequest{})
		if err != nil {
			return fmt.Errorf("unable to get wants: %v", err)
		}

		for i, want := range wants.GetWants() {
			fmt.Printf("%v. %v\n", i, want.GetWant().GetId())
			for _, update := range want.GetUpdates() {
				fmt.Printf("  %v\n", update)
			}
		}

		return nil
	}

	wid, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		_, err = client.AddWant(ctx, &pb.AddWantRequest{
			WantId: wid,
		})
		return err
	default:
		return status.Errorf(codes.InvalidArgument, "%v is not a valid command for handling wants", args[0])
	}
}
