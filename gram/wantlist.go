package main

import (
	"context"
	"fmt"
	"strconv"
	"time"

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

		wtype := pb.WantlistType_ONE_BY_ONE
		dStart := int64(0)
		dEnd := int64(0)
		switch args[1] {
		case "one":
			wtype = pb.WantlistType_ONE_BY_ONE
		case "date":
			wtype = pb.WantlistType_DATE_BOUNDED
			dStartT, err := time.Parse("%y-%M-%d", args[3])
			if err != nil {
				return err
			}
			dStartE, err := time.Parse("%y-%M-%d", args[4])
			if err != nil {
				return err
			}
			dStart = dStartT.UnixNano()
			dEnd = dStartE.UnixNano()
		case "mass":
			wtype = pb.WantlistType_EN_MASSE
		default:
			return status.Errorf(codes.InvalidArgument, "Unknown wantlist type %v", args[1])
		}

		_, err = client.AddWantlist(ctx, &pb.AddWantlistRequest{
			Name:       args[2],
			Type:       wtype,
			Visibility: pb.WantlistVisibility_VISIBLE,
			DateStart:  dStart,
			DateEnd:    dEnd,
		})
		if err != nil {
			return fmt.Errorf("unable to add wantlist: %w", err)
		}

		for _, id := range args[5:] {
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
	} else if args[0] == "deletelist" {
		_, err = client.DeleteWantlist(ctx, &pb.DeleteWantlistRequest{
			Name: args[1],
		})
		if err != nil {
			return fmt.Errorf("unable to delete want: %w", err)
		}
	} else if args[0] == "list" {
		lists, err := client.ListWantlists(ctx, &pb.ListWantlistsRequest{})
		if err != nil {
			return fmt.Errorf("unable to delete want: %w", err)
		}

		for i, list := range lists.GetLists() {
			score := float64(0)
			count := float64(0)
			for _, entry := range list.GetEntries() {
				if entry.GetState() == pb.WantState_PURCHASED || entry.GetState() == pb.WantState_IN_TRANSIT {
					score += float64(entry.GetScore())
					count++
				}
			}
			if count == 0 {
				count = 1
			}
			fmt.Printf("%v. %v [%v] {%v}\n", i, list.GetName(), list.GetType(), score/count)
		}
	} else if args[0] == "type" {
		ntype := pb.WantlistType_TYPE_UNKNOWN
		switch args[2] {
		case "one":
			ntype = pb.WantlistType_ONE_BY_ONE
		case "masse":
			ntype = pb.WantlistType_EN_MASSE
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
	}

	return nil
}
