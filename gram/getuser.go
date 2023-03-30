package main

import (
	"bytes"
	"context"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGetUser() *CLIModule {
	return &CLIModule{
		Command: "user",
		Help:    "Get the stored user details",
		Execute: executeGetUser,
	}
}

func executeGetUser(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileServiceClient(conn)
	user, err := client.GetUser(ctx, &pb.GetUserRequest{})
	if err != nil {
		return err
	}

	buf := bytes.NewBufferString("")
	proto.MarshalText(buf, user)
	fmt.Printf("%v\n", buf.String())
	return nil
}
