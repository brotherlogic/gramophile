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

		printRecord := func(r *pb.Record) {
			fmt.Printf("Record: %v\n", r)
			fmt.Printf("%v\n", resp.GetRecord().GetRelease().GetTitle())

			if resp.GetRecord().GetSaleInfo().GetSaleId() > 0 {
				fmt.Printf("For Sale (%v). Current Price: %v\n", resp.GetRecord().GetSaleInfo().GetSaleId(), resp.GetRecord().GetSaleInfo().GetCurrentPrice())
			}
		}

		for _, record := range resp.GetRecords() {
			printRecord(record)
		}
		if resp.GetRecord() != nil {
			printRecord(resp.GetRecord())
		}
	}
	return err
}
