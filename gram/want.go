package main

import (
	"context"
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

	wid, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return err
	}

	switch args[0] {
	case "add":
		client := pb.NewGramophileEServiceClient(conn)
		_, err = client.AddWant(ctx, &pb.AddWantRequest{
			WantId: wid,
		})
		return err
	case "delete":
		client := pb.NewGramophileEServiceClient(conn)
		_, err = client.DeleteWant(ctx, &pb.DeleteWantRequest{
			WantId: wid,
		})
		return err
	default:
		return status.Errorf(codes.InvalidArgument, "%v is not a valid command for handling wants", args[0])
	}
}
