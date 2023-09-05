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

	if args[0] != "add" {
		return status.Errorf(codes.InvalidArgument, "You can only add wants currently")
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.AddWant(ctx, &pb.AddWantRequest{
		WantId: wid,
	})
	return err
}
