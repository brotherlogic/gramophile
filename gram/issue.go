package main

import (
	"context"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

func GetGetIssue() *CLIModule {
	return &CLIModule{
		Command: "issue",
		Help:    "Get a record with an issue",
		Execute: executeGetIssue,
	}
}

func executeGetIssue(ctx context.Context, args []string) error {

	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)
	records, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithIssue{},
	})
	if err != nil {
		return err
	}

	for _, record := range records.GetRecords() {
		fmt.Printf("%v has: %v\n", record.GetRecord().GetRelease().GetId(), record.GetRecord().GetIssues())
		fmt.Printf("%v\n", record.GetRecord())
	}
	return nil
}
