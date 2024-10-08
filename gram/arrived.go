package main

import (
	"context"
	"strconv"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetArrived() *CLIModule {
	return &CLIModule{
		Command: "arrived",
		Help:    "Items have arrived",
		Execute: executeArrived,
	}
}

func executeArrived(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	t := time.Now()

	if len(args) == 2 {
		if strings.Contains(args[1], "-") {
			t, err = time.Parse("2006-01-02", args[1])
			if err != nil {
				return err
			}
		} else {
			tu, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			t = time.Unix(tu, 0)
		}
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			Arrived: t.UnixNano(),
		},
	})
	return err
}
