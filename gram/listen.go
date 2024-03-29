package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetListen() *CLIModule {
	return &CLIModule{
		Command: "listen",
		Help:    "Listen items",
		Execute: executeListen,
	}
}

func executeListen(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	t := time.Now()

	if len(args) == 0 {
		client := pb.NewGramophileEServiceClient(conn)
		r, err := client.GetRecord(ctx, &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetRecordToListenTo{},
		})
		if err != nil {
			return err
		}
		for _, record := range r.GetRecords() {
			fmt.Printf("%v\n", record.GetRecord().GetRelease().GetInstanceId())
		}
		return nil
	}

	if len(args) == 1 {
		client := pb.NewGramophileEServiceClient(conn)
		r, err := client.GetRecord(ctx, &pb.GetRecordRequest{
			Request: &pb.GetRecordRequest_GetRecordToListenTo{
				GetRecordToListenTo: &pb.GetRecordToListenTo{
					Filter: args[0],
				},
			},
		})
		if err != nil {
			return err
		}
		for _, record := range r.GetRecords() {
			fmt.Printf("%v\n", record.GetRecord().GetRelease().GetInstanceId())
		}
		return nil
	}

	iid, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return err
	}

	newScore := int32(0)
	if len(args) == 2 {
		if strings.Contains(args[1], "-") {
			t, err = time.Parse("2006-01-02", args[1])
			if err != nil {
				return err
			}
		} else {
			tu, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return err
			}
			if tu > 5 {
				t = time.Unix(0, tu)
			} else {
				newScore = int32(tu)
			}
		}
	}

	client := pb.NewGramophileEServiceClient(conn)
	_, err = client.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			ListenTime: t.Unix(),
			NewScore:   newScore,
		},
	})
	return err
}
