package main

import (
	"context"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGetSats() *CLIModule {
	return &CLIModule{
		Command: "stats",
		Help:    "Get collection stats",
		Execute: executeGetStats,
	}
}

func executeGetStats(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	stats, err := client.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		return err
	}

	fmt.Printf("Stats %v\n", stats.GetSaleStats().GetStateCount())
	return nil
}
