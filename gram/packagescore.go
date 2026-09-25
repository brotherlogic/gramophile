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

func GetPackageScore() *CLIModule {
	return &CLIModule{
		Command: "packagescore",
		Help:    "Assign package score to a release",
		Execute: executePackageScore,
	}
}

func runPackageScore(ctx context.Context, client pb.GramophileEServiceClient, args []string) error {
	if len(args) < 2 {
		return status.Errorf(codes.InvalidArgument, "usage: gram packagescore <iid> <0-5>")
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	score, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}

	if score < 0 || score > 5 {
		return status.Errorf(codes.InvalidArgument, "package score must be between 0 and 5, got %d", score)
	}

	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			PackageScore: int32(score),
		},
	})
	return err
}

func executePackageScore(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewGramophileEServiceClient(conn)
	return runPackageScore(ctx, client, args)
}
