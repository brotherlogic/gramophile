package main

import (
	"context"
	"fmt"
	"os"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/golang/protobuf/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGetConfig() *CLIModule {
	return &CLIModule{
		Command: "config",
		Help:    "Get the user config",
		Execute: executeGetConfig,
	}
}

func executeGetConfig(ctx context.Context, _ []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	user, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		return err
	}

	if user.GetUser().GetConfig() == nil {
		fmt.Printf("%v\n", &pb.GramophileConfig{})
	} else {
		proto.MarshalText(os.Stdout, user.GetUser().GetConfig())
	}

	return nil
}
