package main

import (
	"context"
	"flag"
	"fmt"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func GetOrganisation() *CLIModule {
	return &CLIModule{
		Command: "org",
		Help:    "Get orgs",
		Execute: executeOrg,
	}
}

func getArtist(r *pbd.Release) string {
	if len(r.GetArtists()) == 0 {
		return "NO_ARTIST"
	}
	artist := r.GetArtists()[0].GetName()
	for _, art := range r.GetArtists()[1:] {
		artist += ", " + art.GetName()
	}
	return artist
}

func resolvePlacement(ctx context.Context, client pb.GramophileEServiceClient, p *pb.Placement, debug bool) (string, error) {
	r, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				InstanceId: p.GetIid(),
			}}})

	if err != nil {
		return "", err
	}

	str := ""
	for _, record := range r.GetRecords() {
		str += fmt.Sprintf("%v - %v [%v / %v] %v",
			getArtist(record.GetRecord().GetRelease()),
			record.GetRecord().GetRelease().GetTitle(),
			p.GetWidth(), record.GetRecord().GetWidth(), time.Unix(0, record.GetRecord().GetRelease().GetDateAdded()))
		if debug {
			str += fmt.Sprintf(" {%v - %v (%v)}", p.GetOriginalIndex(), p.GetObservations(), p.GetSpace())
		}
	}

	return str, nil
}

func executeOrg(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to dial gramophile: %w", err)
	}

	if len(args) > 1 {
		orgFlags := flag.NewFlagSet("orgflags", flag.ExitOnError)
		name := orgFlags.String("org", "", "The name of the organisation")
		slot := orgFlags.Int("slot", 1, "The slot to print")
		debug := orgFlags.Bool("debug", false, "Include debug info")
		hash := orgFlags.String("hash", "", "The hash of org snapshot")
		//space := orgFlags.String("space", "")

		if err := orgFlags.Parse(args); err == nil {
			client := pb.NewGramophileEServiceClient(conn)
			r, err := client.GetOrg(ctx, &pb.GetOrgRequest{
				OrgName: *name,
				Hash:    *hash,
			})
			if err != nil {
				return fmt.Errorf("unable to get org: %w", err)
			}

			fmt.Printf("%v -> %v\n", time.Unix(0, r.GetSnapshot().GetDate()), r.GetSnapshot().GetHash())

			if len(r.GetSnapshot().GetPlacements()) == 0 {
				return status.Errorf(codes.InvalidArgument, "org %v has no elements", *name)
			}

			currSlot := 0
			currShelf := ""
			cslot := int32(0)
			totalWidth := float32(0)
			for i, placement := range r.GetSnapshot().GetPlacements() {
				if placement.GetSpace() != currShelf || placement.GetUnit() != cslot {
					currShelf = placement.GetSpace()
					cslot = placement.GetUnit()
					currSlot++
				}

				if currSlot == *slot || *slot == -1 {
					pstr, err := resolvePlacement(ctx, client, placement, *debug)
					if err != nil {
						return fmt.Errorf("unable to place %v -> %w", placement.GetIid(), err)
					}
					fmt.Printf("%v. [%v-%v] %v\n", i, placement.GetSpace(), placement.GetUnit(), pstr)
					totalWidth += placement.GetWidth()
				}
			}
			fmt.Printf("Total Width = %v\n", totalWidth)
			return nil
		}
	}
	return err
}
