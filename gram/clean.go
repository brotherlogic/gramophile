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

func GetClean() *CLIModule {
	return &CLIModule{
		Command: "clean",
		Help:    "Clean items",
		Execute: executeClean,
	}
}

func executeClean(ctx context.Context, args []string) error {
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
			t = time.Unix(0, tu)
		}
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			CleanTime: t.Unix(),
		},
	})
	return err
}
