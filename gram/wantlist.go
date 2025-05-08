package main

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetWantlist() *CLIModule {
	return &CLIModule{
		Command: "wantlist",
		Help:    "wantlist",
		Execute: executeWantlist,
	}
}

func executeWantlist(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}

	client := pb.NewGramophileEServiceClient(conn)

	wantlist, err := client.GetWantlist(ctx, &pb.GetWantlistRequest{Name: args[0]})
	if err != nil {
		return err
	}

	total := float64(0)
	count := float64(0)
	for _, entry := range wantlist.GetList().GetEntries() {
		if entry.GetScore() > 0 {
			total += float64(entry.GetScore())
			count++
		}
	}

	fmt.Printf("List: %v (%v) [%v (%v)]\n", wantlist.GetList().GetName(), wantlist.GetList().GetType(), wantlist.GetList().GetActive(), total/count)
	fmt.Printf("Updated: %v\n", time.Unix(0, wantlist.GetList().GetLastUpdatedTimestamp()))
	for _, entry := range wantlist.GetList().GetEntries() {
		fmt.Printf("  [%v] %v - %v (%v) [%v]\n", entry.GetId(), entry.GetArtist(), entry.GetTitle(), entry.GetState(), entry.GetScore())
	}

	return nil
}
