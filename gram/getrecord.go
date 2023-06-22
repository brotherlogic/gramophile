package main

import (
	"context"
	"flag"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetGetRecord() *CLIModule {
	return &CLIModule{
		Command: "get",
		Help:    "Get a record from the db",
		Execute: executeGetRecord,
	}
}

func executeGetRecord(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	idSet := flag.NewFlagSet("ids", flag.ExitOnError)
	var id = idSet.Int("id", 0, "Id of record to get")
	var iid = idSet.Int("iid", 0, "IId of record to get")
	if err := idSet.Parse(args); err == nil {

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetRecord(ctx, &pb.GetRecordRequest{Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				InstanceId: int64(*iid),
				ReleaseId:  int64(*id),
			},
		}})
		if err != nil {
			return err
		}

		for _, record := range resp.GetRecords() {
			fmt.Printf("%v\n", record)
		}
		if resp.GetRecord() != nil {
			fmt.Printf("%v\n", resp.GetRecord())
		}
	}
	return err
}
