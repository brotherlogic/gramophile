package main

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func GetWantlist() *CLIModule {
	return &CLIModule{
		Command: "wantlist",
		Help:    "wantlist",
		Execute: executeWantlist,
	}
}

func executeWantlist(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	if args[0] != "add" {
		return status.Errorf(codes.InvalidArgument, "Currently only support adding wantslist")
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.AddWantlist(ctx, &pb.AddWantlistRequest{
		Name: args[1],
	})
	if err != nil {
		return err
	}

	for _, id := range args[2:] {
		wid, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			return err
		}

		_, err = client.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
			AddId: wid,
		})
		if err != nil {
			return fmt.Errorf("unable to update wantlist: %w", err)
		}
	}

	return nil
}
