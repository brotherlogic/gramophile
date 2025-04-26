package main

import (
	"context"
	"flag"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetAdd() *CLIModule {
	return &CLIModule{
		Command: "add",
		Help:    "Add item",
		Execute: executeAdd,
	}
}

func executeAdd(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	flgs := flag.NewFlagSet("add", flag.ExitOnError)
	id := flgs.Int64("id", -1, "Release id of the record being added")
	price := flgs.Float64("price", -1, "The price of the record")
	location := flgs.String("location", "", "The purchase location")

	err = flgs.Parse(args)
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.AddRecord(ctx, &pb.AddRecordRequest{
		Id:       *id,
		Price:    int32(*price * 100),
		Location: *location,
	})
	return err
}
