package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brotherlogic/goserver/utils"

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
}
