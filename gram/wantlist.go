package main

import (
	"context"
	"fmt"
	"strconv"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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
			Name:       args[1],
			Type:       pb.WantlistType_ONE_BY_ONE,
			Visibility: pb.WantlistVisibility_VISIBLE,
		})
		if err != nil {
			return fmt.Errorf("unable to add wantlist: %w", err)
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
	} else if args[0] == "list" {
		lists, err := client.ListWantlists(ctx, &pb.ListWantlistsRequest{})
		if err != nil {
			return fmt.Errorf("unable to delete want: %w", err)
		}

		for i, list := range lists.GetLists() {
			fmt.Printf("%v. %v [%v]\n", i, list.GetName(), list.GetType())
		}
	} else if args[0] == "type" {
		ntype := pb.WantlistType_TYPE_UNKNOWN
		switch args[2] {
		case "one":
			ntype = pb.WantlistType_ONE_BY_ONE
		default:
			return status.Errorf(codes.InvalidArgument, "%v is not a known type [one]", args[2])
		}

		_, err = client.UpdateWantlist(ctx, &pb.UpdateWantlistRequest{
			Name:    args[1],
			NewType: ntype,
		})
		if err != nil {
			return fmt.Errorf("unable to delete want: %w", err)
		}
	} else {
		wantlist, err := client.GetWantlist(ctx, &pb.GetWantlistRequest{Name: args[0]})
		if err != nil {
			return err
		}

		fmt.Printf("List: %v (%v)\n", wantlist.GetList().GetName(), wantlist.GetList().GetType())
		for _, entry := range wantlist.GetList().GetEntries() {
			fmt.Printf("  [%v] %v - %v (%v)\n", entry.GetId(), entry.GetArtist(), entry.GetTitle(), entry.GetState())
		}
	}

	return nil
}
