package main

import (
	"context"
	"strconv"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetWidth() *CLIModule {
	return &CLIModule{
		Command: "width",
		Help:    "Width",
		Execute: executeWidth,
	}
}

func executeWidth(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	width, err := strconv.ParseFloat(args[1], 10)
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			Width: float32(width),
		},
	})
	return err
}
