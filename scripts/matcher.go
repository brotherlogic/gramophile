package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/brotherlogic/goserver/utils"
	"github.com/golang/protobuf/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	pbg "github.com/brotherlogic/gramophile/proto"
	pbro "github.com/brotherlogic/recordsorganiser/proto"
)

func main() {
	ctx, cancel := utils.ManualContext("syncing", time.Hour)
	defer cancel()

	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("No home dir: %v", err)
	}

	text, err := ioutil.ReadFile(fmt.Sprintf("%v/.gramophile", dirname))
	if err != nil {
		log.Fatalf("Failed to read token: %v", err)
	}

	user := &pbg.GramophileAuth{}
	err = proto.UnmarshalText(string(text), user)
	if err != nil {
		log.Fatalf("Failed to unmarshal: %v", err)
	}

	ctx = metadata.AppendToOutgoingContext(ctx, "auth-token", user.GetToken())

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
	if err != nil {
		log.Fatalf("Unable to get org: %v", err)
	}
	fmt.Printf("Found %v records in %v\n", len(r.GetSnapshot().GetPlacements()), os.Args[2])

	if len(r.GetSnapshot().GetPlacements()) != len(records.GetLocations()[0].GetReleasesLocation()) {
		fmt.Printf("MISMATCH: Different number of entries in each\n")
		return
	}
}
