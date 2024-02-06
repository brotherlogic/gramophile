package main

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetWeight() *CLIModule {
	return &CLIModule{
		Command: "weight",
		Help:    "weight",
		Execute: executeWeight,
	}
}

func executeWeight(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to reach gramophile: %w", err)
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	weight, err := strconv.ParseFloat(args[1], 10)
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			Weight: float32(weight),
		},
	})
	return err
}
