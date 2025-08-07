package main

import (
	"context"
	"flag"
	"fmt"
	"sort"
	"strconv"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func GetWant() *CLIModule {
	return &CLIModule{
		Command: "want",
		Help:    "want",
		Execute: executeWant,
	}
}

func executeWant(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	client := pb.NewGramophileEServiceClient(conn)

	// This is just a list
	if len(args) == 0 {
		wants, err := client.GetWants(ctx, &pb.GetWantsRequest{})
		if err != nil {
			return fmt.Errorf("unable to get wants: %v", err)
		}

		for i, want := range wants.GetWants() {
			fmt.Printf("%v. %v [%v]\n", i, want.GetWant().GetId(), want.GetWant().GetState())

			sort.SliceStable(want.Updates, func(i, j int) bool {
				return want.GetUpdates()[i].Date < want.GetUpdates()[j].Date
			})

			for _, update := range want.GetUpdates() {
				fmt.Printf("  %v - %v\n", time.Unix(0, update.GetDate()), update)
			}
		}

		return nil
	}

	switch args[0] {
	case "add":
		wid, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return err
		}

		_, err = client.AddWant(ctx, &pb.AddWantRequest{
			WantId: wid,
		})
		return err
	case "get":
		flgs := flag.NewFlagSet("get", flag.ExitOnError)
		id := flgs.Int64("id", -1, "Release id of the record being added")
		debug := flgs.Bool("debug", false, "Displau the raw proto")

		err = flgs.Parse(args[1:])
		if err != nil {
			return err
		}

		wants, err := client.GetWants(ctx, &pb.GetWantsRequest{ReleaseId: *id, IncludeUpdates: true})
		if err != nil {
			return err
		}
		want := wants.GetWants()[0]
		fmt.Printf("%v [%v] (%v)\n", want.GetWant().GetId(), want.GetWant().GetState(), want.GetWant().GetIntendedState())
		fmt.Printf("Last updated %v\n\n", time.Unix(0, want.GetWant().GetWantAddedDate()))

		sort.SliceStable(want.Updates, func(i, j int) bool {
			return want.GetUpdates()[i].Date < want.GetUpdates()[j].Date
		})

		for _, update := range want.GetUpdates() {
			fmt.Printf("  %v - %v\n", time.Unix(0, update.GetDate()), update)
		}

		if *debug {
			fmt.Printf("\n\n%v\n", want)
		}

		return nil
	default:
		return status.Errorf(codes.InvalidArgument, "%v is not a valid command for handling wants", args[0])
	}
}
