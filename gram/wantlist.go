package main

import (
	"context"
	"fmt"
	"strconv"

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

	if args[0] == "add" {

		_, err = client.AddWantlist(ctx, &pb.AddWantlistRequest{
			Name: args[1],
		})
		if err != nil {
			return err
		}

		for _, id := range args[2:] {
			wid, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return err
			}

			_, err = client.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
				Name:  args[1],
				AddId: wid,
			})
			if err != nil {
				return fmt.Errorf("unable to update wantlist: %w", err)
			}
		}
	} else if args[0] == "delete" {
		for _, id := range args[2:] {
			wid, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				return err
			}

			_, err = client.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
				Name:     args[1],
				DeleteId: wid,
			})
			if err != nil {
				return fmt.Errorf("unable to delete want: %w", err)
			}
		}
	} else {
		wantlist, err := client.GetWantlist(ctx, &pb.GetWantlistRequest{Name: args[0]})
		if err != nil {
			return err
		}

		fmt.Printf("List: %v\n", wantlist.GetList().GetName())
		for _, entry := range wantlist.GetList().GetEntries() {
			fmt.Printf("  [%v] %v - %v ", entry.GetId(), entry.GetArtist(), entry.GetTitle())
		}
	}

	return nil
}
