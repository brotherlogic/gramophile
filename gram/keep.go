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

func GetKeep() *CLIModule {
	return &CLIModule{
		Command: "keep",
		Help:    "Keep stuff",
		Execute: executeKeep,
	}
}

func executeKeep(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	var keepState pb.KeepStatus
	switch args[1] {
	case "none":
		keepState = pb.KeepStatus_NO_KEEP
	case "digital":
		keepState = pb.KeepStatus_DIGITAL_KEEP
	case "keep":
		keepState = pb.KeepStatus_KEEP
	case "mintup":
		keepState = pb.KeepStatus_MINT_UP_KEEP
	case "reset":
		keepState = pb.KeepStatus_RESET
	default:
		return status.Errorf(codes.FailedPrecondition, "%v is not a valid keep state (none, digital, keep, mintup)", args[1])
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			Keep: keepState,
		},
	})
	return err
}
