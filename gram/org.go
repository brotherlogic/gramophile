package main

import (
	"context"
	"flag"
	"fmt"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetOrganisation() *CLIModule {
	return &CLIModule{
		Command: "org",
		Help:    "Get orgs",
		Execute: executeOrg,
	}
}

func getArtist(r *pbd.Release) string {
	return "NEED_TO_IMPLEMENT_ARTIST_RETRIEVAL"
}

func resolvePlacement(ctx context.Context, client pb.GramophileEServiceClient, p *pb.Placement) (string, error) {
	r, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{
				InstanceId: p.GetIid(),
			}}})

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%v - %v", getArtist(r.GetRecord().GetRelease()), r.GetRecord().GetRelease().GetTitle()), nil
}

func executeOrg(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to dial gramophile: %w", err)
	}

	if len(args) == 1 {
		orgFlags := flag.NewFlagSet("orgflags", flag.ExitOnError)
		name := orgFlags.String("org", "", "The name of the organisation")
		slot := orgFlags.Int("slot", -1, "The slot to print")

		if err := orgFlags.Parse(args); err == nil {
			client := pb.NewGramophileEServiceClient(conn)
			r, err := client.GetOrg(ctx, &pb.GetOrgRequest{
				OrgName: *name,
			})
			if err != nil {
				return fmt.Errorf("unable to get org: %w", err)
			}
			currSlot := 0
			currShelf := ""
			for _, placement := range r.GetSnapshot().GetPlacements() {
				if placement.GetSpace() != currShelf {
					currShelf = placement.GetSpace()
					currSlot++
				}

				if currSlot == *slot {
					pstr, err := resolvePlacement(ctx, client, placement)
					if err != nil {
						return err
					}
					fmt.Printf("%v\n", pstr)
				}
			}
			return nil
		}
	}
	return err
}
