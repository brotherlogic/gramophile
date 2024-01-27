package main

import (
	"context"
	"flag"
	"fmt"
	"time"

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
	var history = idSet.Bool("history", false, "Whether to get the history")
	var debug = idSet.Bool("debug", false, "Show debug stuff")
	if err := idSet.Parse(args); err == nil {

		client := pb.NewGramophileEServiceClient(conn)
		resp, err := client.GetRecord(ctx, &pb.GetRecordRequest{
			IncludeHistory: *history,
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					InstanceId: int64(*iid),
					ReleaseId:  int64(*id),
				},
			}})
		if err != nil {
			return err
		}

		printRecord := func(r *pb.RecordResponse, debug bool) {
			fmt.Printf("%v [%v]\n", r.GetRecord().GetRelease().GetTitle(), r.GetRecord().GetRelease().GetInstanceId())
			fmt.Printf("%v / %v\n", r.GetRecord().GetWidth(), r.GetRecord().GetWeight())
			fmt.Printf("Sale: %v -> %v [%v]\n", r.GetRecord().GetSaleId(), time.Unix(0, r.GetSaleInfo().GetLastPriceUpdate()), r.GetSaleInfo().GetCurrentPrice().GetValue())
			for _, update := range r.GetSaleInfo().GetUpdates() {
				fmt.Printf("  %v -> %v\n", time.Unix(0, update.GetDate()), update.GetSetPrice().GetValue())
			}

			fmt.Printf("Current Price: $%.2f\n", float32(r.GetSaleInfo().GetCurrentPrice().GetValue())/100.0)
			fmt.Printf("Median Price:  $%.2f\n", float32(r.GetRecord().GetMedianPrice().GetValue())/100.0)
			fmt.Printf("Low Price:     $%.2f\n", float32(r.GetRecord().GetLowPrice().GetValue())/100.0)
			fmt.Printf("Median Reached on %v\n", time.Unix(0, r.GetSaleInfo().GetTimeAtMedian()))
			fmt.Printf("Last Updated on %v\n", time.Unix(0, r.GetRecord().GetLastUpdateTime()))
			fmt.Printf("Sale Updated on %v\n", time.Unix(0, r.GetSaleInfo().GetLastPriceUpdate()))

			if debug {
				fmt.Printf("%v\n", r.GetRecord())
			}

			for _, update := range r.GetUpdates() {
				fmt.Printf(" %v -> %v\n", update.GetDate(), update.GetExplanation())
			}
		}

		for _, record := range resp.GetRecords() {
			printRecord(record, *debug)
		}
		if resp.GetRecordResponse() != nil {
			printRecord(resp.GetRecordResponse(), *debug)
		}
	}
	return err
}
