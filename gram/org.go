package main

import (
	"context"
	"fmt"

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

func executeOrg(ctx context.Context, args []string) error {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("unable to dial gramophile: %w", err)
	}

	if len(args) == 1 {
		client := pb.NewGramophileEServiceClient(conn)
		r, err := client.GetOrg(ctx, &pb.GetOrgRequest{
			OrgName: args[0],
		})
		if err != nil {
			return fmt.Errorf("unable to get org: %w", err)
		}
		for _, placement := range r.GetSnapshot().GetPlacements() {
			if placement.GetUnit() == 1 {
				fmt.Printf("%v\n", placement)
			}
		}
		return nil
	}
	return fmt.Errorf("need to supply org name")
}
