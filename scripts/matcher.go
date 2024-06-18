package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brotherlogic/goserver/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pbg "github.com/brotherlogic/gramophile/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
)

func main() {
	ctx, cancel := utils.ManualContext("syncing", time.Hour)
	defer cancel()

	conn, err := utils.LFDialServer(ctx, "recordsorganiser")
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}
	defer conn.Close()

	client := pbro.NewOrganiserServiceClient(conn)

	records, err := client.GetOrganisation(ctx, &pbro.GetOrganisationRequest{Locations: []*pbro.Location{&pbro.Location{Name: os.Args[1]}}})
	if err != nil {
		log.Fatalf("Bad org get: %v", err)
	}

	fmt.Printf("Found %v records in %v\n", len(records.GetLocations()[0].GetReleasesLocation()), os.Args[1])

	conn2, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("unable to dial gramophile: %w", err)
	}
	kclient := pbg.NewGramophileEServiceClient(conn2)
	r, err := kclient.GetOrg(ctx, &pbg.GetOrgRequest{
		OrgName: os.Args[2],
	})
	log.Printf("Found %v records in %v\n", len(r.GetSnapshot().GetPlacements()))

}
