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

func getTitle(ctx context.Context, client pb.GramophileEServiceClient, iid int64) string {
	res, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				InstanceId: iid,
			},
		},
	})
	if err != nil || len(res.GetRecords()) == 0 {
		return fmt.Sprintf("Unknown (%v)", iid)
	}
	return res.GetRecords()[0].GetRecord().GetRelease().GetTitle()
}

func getTitleFromRelease(ctx context.Context, client pb.GramophileEServiceClient, releaseId int64) string {
	res, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				ReleaseId: releaseId,
			},
		},
	})
	if err != nil || len(res.GetRecords()) == 0 {
		return fmt.Sprintf("Unknown (%v)", releaseId)
	}
	return res.GetRecords()[0].GetRecord().GetRelease().GetTitle()
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

		title := getTitleFromRelease(ctx, client, int64(*id))

		for _, location := range res.GetLocations() {
			fmt.Printf("%v is in %v, Slot %v:\n\n", title, location.GetLocationName(), location.GetSlot())

			for i := len(location.GetBefore()) - 1; i >= 0; i-- {
				b := location.GetBefore()[i]
				fmt.Printf("%v\n", getTitle(ctx, client, b.GetIid()))
			}

			fmt.Printf("%v\n", title)

			for _, a := range location.GetAfter() {
				fmt.Printf("%v\n", getTitle(ctx, client, a.GetIid()))
			}
			fmt.Printf("\n")
		}
	}

	return nil
}
