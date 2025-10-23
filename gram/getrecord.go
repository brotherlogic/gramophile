package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	pbgd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"
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
	var sid = idSet.Int("sid", 0, "Sale ID")
	var lid = idSet.Int("lid", 0, "Label ID")

	var minmed = idSet.Int64("minmed", 0, "Minumum median seconds")
	var history = idSet.Bool("history", false, "Whether to get the history")
	var debug = idSet.Bool("debug", false, "Show debug stuff")
	var mintup = idSet.Bool("mintup", false, "Get records to mint up on")

	var binary = idSet.Bool("binary", false, "Print a binary dump of the returned records")

	if err := idSet.Parse(args); err == nil {
		client := pb.NewGramophileEServiceClient(conn)

		if *mintup {
			resp, err := client.GetRecord(ctx, &pb.GetRecordRequest{
				Request: &pb.GetRecordRequest_GetRecordsMintUp{
					GetRecordsMintUp: true,
				},
			})
			if err != nil {
				return err
			}

			for _, r := range resp.GetRecords() {
				fmt.Printf("%v [%v]\n", r.GetRecord().GetRelease().GetTitle(), r.GetRecord().GetRelease().GetInstanceId())
			}

			return nil
		}

		if *minmed > 0 {
			sales, err := client.GetSale(ctx, &pb.GetSaleRequest{MinMedian: *minmed})
			if err != nil {
				return err
			}

			for _, sale := range sales.GetSales() {
				lowdate := time.Now().Add(time.Hour).UnixNano()
				for _, hist := range sale.GetUpdates() {
					if hist.GetSetPrice().GetValue() == sale.GetLowPrice().GetValue() {
						if hist.GetDate() < lowdate {
							lowdate = hist.GetDate()
						}
					}
				}
				if time.Unix(0, lowdate).Before(time.Now()) {
					if sale.GetSaleState() == pbgd.SaleStatus_FOR_SALE {
						fmt.Printf("%v - %v\n", time.Since(time.Unix(0, lowdate)), sale.GetReleaseId())
					}
				}
			}
		}

		if *sid > 0 {
			sale, err := client.GetSale(ctx, &pb.GetSaleRequest{Id: int64(*sid)})
			if err != nil {
				return err
			}
			fmt.Printf("%v\n", sale)
			return nil
		}

		resp, err := client.GetRecord(ctx, &pb.GetRecordRequest{
			IncludeHistory: *history,
			Request: &pb.GetRecordRequest_GetRecordWithId{
				GetRecordWithId: &pb.GetRecordWithId{
					InstanceId: int64(*iid),
					ReleaseId:  int64(*id),
					LabelId:    int32(*lid),
				},
			}})
		if err != nil {
			return err
		}

		printRecord := func(r *pb.RecordResponse, debug bool) {
			fmt.Printf("%v [%v / %v] %v -> %v\n", r.GetRecord().GetRelease().GetTitle(), r.GetRecord().GetRelease().GetInstanceId(), r.GetRecord().GetRelease().GetMasterId(), time.Unix(0, r.GetRecord().GetRelease().GetDateAdded()), r.GetRecord().GetRefreshId())
			fmt.Printf("In folder %v [FR %v / R %v]\n", r.GetRecord().GetRelease().GetFolderId(), time.Unix(r.GetRecord().GetEarliestReleaseDate(), 0), time.Unix(r.GetRecord().GetRelease().GetReleaseDate(), 0))
			fmt.Printf("%v\n", r.GetRecord().GetKeepStatus())
			fmt.Printf("Sale: %v - (%v) > %v [%v / %v]\n",
				r.GetRecord().GetSaleId(),
				r.GetSaleInfo().GetSaleState(),
				time.Unix(0, r.GetSaleInfo().GetLastPriceUpdate()),
				r.GetSaleInfo().GetCurrentPrice().GetValue(),
				r.GetSaleInfo().GetRefreshId())
			for _, update := range r.GetSaleInfo().GetUpdates() {
				fmt.Printf("  %v -> %v %v\n", time.Unix(0, update.GetDate()), update.GetSetPrice().GetValue(), update.GetMotivation())
			}
			fmt.Printf("Width: %v\n", r.GetRecord().GetWidth())

			fmt.Printf("Current Price: $%.2f\n", float32(r.GetSaleInfo().GetCurrentPrice().GetValue())/100.0)
			fmt.Printf("Median Price:  $%.2f\n", float32(r.GetRecord().GetMedianPrice().GetValue())/100.0)
			fmt.Printf("Low Price:     $%.2f\n", float32(r.GetRecord().GetLowPrice().GetValue())/100.0)
			fmt.Printf("High Price:    $%.2f\n", float32(r.GetRecord().GetHighPrice().GetValue())/100.0)
			fmt.Printf("Median Reached on %v\n", time.Unix(0, r.GetSaleInfo().GetTimeAtMedian()))
			fmt.Printf("Low Reached on %v\n", time.Unix(0, r.GetSaleInfo().GetTimeAtLow()))
			fmt.Printf("Last Updated on %v\n", time.Unix(0, r.GetRecord().GetLastUpdateTime()))
			fmt.Printf("Stats Updated on %v\n", time.Unix(0, r.GetRecord().GetLastStatRefresh()))
			fmt.Printf("Sale Updated on %v\n", time.Unix(0, r.GetSaleInfo().GetLastPriceUpdate()))
			fmt.Printf("ERD Updated on %v\n", time.Unix(0, r.GetRecord().GetLastEarliestReleaseUpdate()))
			fmt.Printf("Classified to %v\n", r.GetCategory())
			fmt.Printf("There are %v digital versions\n", len(r.GetRecord().GetDigitalIds()))
			fmt.Printf("ERD: %v\n", r.GetRecord().GetEarliestReleaseDate())

			if debug {
				fmt.Printf("%v\n", r.GetRecord())
			}

			for _, update := range r.GetUpdates() {
				fmt.Printf(" %v -> %v\n", update.GetDate(), update)
			}
		}

		if *binary {
			rs := &pb.RecordSet{}
			for _, r := range resp.GetRecords() {
				rs.Records = append(rs.Records, r.GetRecord())
			}
			data, err := proto.Marshal(rs)
			if err != nil {
				log.Fatalf("Unable to marshal records: %v", err)
			}
			fmt.Printf("Result: %x\n", data)
		} else {
			for _, record := range resp.GetRecords() {
				printRecord(record, *debug)
			}
		}
	}
	return err
}
