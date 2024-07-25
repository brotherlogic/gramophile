package main

import (
	"context"
	"strconv"
	
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetFolder() *CLIModule {
	return &CLIModule{
		Command: "folder",
		Help:    "Move the record",
		Execute: executeFolder,
	}
}

func executeFolder(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	folderId, err := strconv.ParseInt(args[1], 10, 32)
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			NewFolder: int32(folderId),
		},
	})
	return err
}
