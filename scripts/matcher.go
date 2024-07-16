package main

import (
	"context"
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

	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
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

	result, err := getDiff(ctx)
	if err != nil {
		log.Fatalf("err")
	}

	if result == "" {
		result = "All Clear"
	}

	password, err := os.ReadFile(fmt.Sprintf("%v/.ghb", dirname))
	if err != nil {
		log.Fatalf("Can't read token: %v", err)
	}
	client, err := ghbclient.GetClientExternal(string(password))
	if err != nil {
		log.Fatalf("Bad client: %v", err)
	}
	_, err = client.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
		User:  "brotherlogic",
		Repo:  "gramophile",
		Title: "Sorting mismtach",
		Body:  result,
	})
	if err != nil {
		log.Fatalf("Bad create: %v", err)
	}
}

func allowed(iid int64) bool {
	for _, val := range []int64{
		19867938,
		19867939,
		19867401,
		243361896,
		1473580069,
		19867473,
		115735835} {
		if val == iid {
			return true
		}
	}

	return false
}

func getDiff(ctx context.Context) (string, error) {
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

	//fmt.Printf("Found %v records in %v\n", len(records.GetLocations()[0].GetReleasesLocation()), os.Args[1])

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
	//fmt.Printf("Found %v records in %v\n", len(r.GetSnapshot().GetPlacements()), os.Args[2])

	if len(r.GetSnapshot().GetPlacements()) != len(records.GetLocations()[0].GetReleasesLocation()) {
		return "MISMATCH: Different number of entries in each", nil
	}

	for i, p := range r.GetSnapshot().GetPlacements() {
		if p.GetIid() != int64(records.GetLocations()[0].GetReleasesLocation()[i].GetInstanceId()) {
			if !allowed(p.GetIid()) {
				return fmt.Sprintf("MISMATCH: %v: %v vs %v\n", i, p.GetIid(), records.GetLocations()[0].GetReleasesLocation()[i].GetInstanceId()), nil
			}
		}
	}

	return "", nil
}
