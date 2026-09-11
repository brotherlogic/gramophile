package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	pbgd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetSales() *CLIModule {
	return &CLIModule{
		Command: "sales",
		Help:    "List active sales",
		Execute: executeSales,
	}
}

func printSale(s *pb.SaleInfo) {
	priceStr := fmt.Sprintf("%.2f %s", float64(s.GetCurrentPrice().GetValue())/100.0, s.GetCurrentPrice().GetCurrency())
	listed := "Unknown"
	if s.GetListedDate() > 0 {
		listed = time.Unix(0, s.GetListedDate()).Format("2006-01-02")
	} else if s.GetTimeCreated() > 0 {
		listed = time.Unix(0, s.GetTimeCreated()).Format("2006-01-02")
	}

	fmt.Printf("Sale %d | Release %d | State: %v | Price: %s | Condition: %s | Listed: %s\n",
		s.GetSaleId(),
		s.GetReleaseId(),
		s.GetSaleState(),
		priceStr,
		s.GetCondition(),
		listed,
	)
}

func executeSales(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	salesSet := flag.NewFlagSet("sales", flag.ExitOnError)
	all := salesSet.Bool("all", false, "Show all sales including inactive/sold")
	sid := salesSet.Int64("sid", 0, "Show specific sale ID")

	if err := salesSet.Parse(args); err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)

	if *sid > 0 {
		resp, err := client.GetSale(ctx, &pb.GetSaleRequest{Id: *sid})
		if err != nil {
			return err
		}
		for _, s := range resp.GetSales() {
			printSale(s)
		}
		return nil
	}

	resp, err := client.GetSale(ctx, &pb.GetSaleRequest{MinMedian: -1})
	if err != nil {
		return err
	}

	count := 0
	for _, sale := range resp.GetSales() {
		if *all || sale.GetSaleState() == pbgd.SaleStatus_FOR_SALE {
			count++
			printSale(sale)
		}
	}

	fmt.Printf("\nTotal: %d sales listed\n", count)
	return nil
}
