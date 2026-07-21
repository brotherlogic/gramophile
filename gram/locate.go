package main

import (
	"context"
	"flag"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetLocate() *CLIModule {
	return &CLIModule{
		Command: "locate",
		Help:    "Locate a record by ID",
		Execute: executeLocate,
	}
}

func calculatePercentage(beforeCount, afterCount int) float64 {
	total := beforeCount + afterCount + 1
	targetIndex := beforeCount + 1
	return float64(targetIndex) / float64(total) * 100.0
}

func formatLocationOutput(location *pb.Location) string {
	percentage := calculatePercentage(len(location.GetBefore()), len(location.GetAfter()))
	out := fmt.Sprintf("%v is in %v, Slot %v (%.0f %%):\n\n", location.GetRecord(), location.GetLocationName(), location.GetSlot(), percentage)

	for i := len(location.GetBefore()) - 1; i >= 0; i-- {
		b := location.GetBefore()[i]
		out += fmt.Sprintf("%v\n", b.GetRecord())
	}

	out += fmt.Sprintf("%v\n", location.GetRecord())

	for _, a := range location.GetAfter() {
		out += fmt.Sprintf("%v\n", a.GetRecord())
	}
	out += "\n"
	return out
}

func executeLocate(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	idSet := flag.NewFlagSet("ids", flag.ExitOnError)
	var id = idSet.Int("id", 0, "Id of record to locate")

	if err := idSet.Parse(args); err == nil {
		client := pb.NewGramophileEServiceClient(conn)

		res, err := client.LocateRecord(ctx, &pb.LocateRecordRequest{
			ReleaseId: int64(*id),
		})
		if err != nil {
			return err
		}

		for _, location := range res.GetLocations() {
			fmt.Print(formatLocationOutput(location))
		}
	}

	return nil
}
