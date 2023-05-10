package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
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

func executeGetConfig(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)

	if len(args) == 0 {
		user, err := client.GetUser(ctx, &pb.GetUserRequest{})
		if err != nil {
			return err
		}

		if user.GetUser().GetConfig() == nil {
			fmt.Printf("%v\n", &pb.GramophileConfig{})
		} else {
			proto.MarshalText(os.Stdout, user.GetUser().GetConfig())
		}
	} else {
		gconfig := &pb.GramophileConfig{}
		data, err := ioutil.ReadFile(args[0])
		if err != nil {
			log.Fatalf("Unable to read file: %v", err)
		}
		err = proto.Unmarshal(data, gconfig)
		if err != nil {
			log.Fatalf("Problem parsing config file: %v", err)
		}

		_, err = client.SetConfig(ctx, &pb.SetConfigRequest{Config: gconfig})
		if err != nil {
			log.Fatalf("Error setting config: %v", err)
		}
	}

	return nil
}
