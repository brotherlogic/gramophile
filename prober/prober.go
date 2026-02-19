package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	ghb_client "github.com/brotherlogic/githubridge/client"
	ghpb "github.com/brotherlogic/githubridge/proto"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type prober interface {
	runProbe(ctx context.Context, client pb.GramophileEServiceClient, iclient pb.GramophileServiceClient) error
	getName() string
	getFrequency() time.Duration
}

var (
	probers = []prober{&onboarding{}}
)

func GetContextKey(ctx context.Context) (string, error) {
	md, found := metadata.FromIncomingContext(ctx)
	if found {
		if _, ok := md["auth-token"]; ok {
			idt := md["auth-token"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	md, found = metadata.FromOutgoingContext(ctx)
	if found {
		if _, ok := md["auth-token"]; ok {
			idt := md["auth-token"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	return "", status.Errorf(codes.NotFound, "Could not extract token from incoming or outgoing")
}

func runProbe(ctx context.Context, probe prober) error {
	// Build the clients
	conne, err := grpc.NewClient("gramophile.gramophile:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}
	eclient := pb.NewGramophileEServiceClient(conne)

	conni, err := grpc.NewClient("gramophile.gramophile:8083")
	if err != nil {
		return err
	}
	iclient := pb.NewGramophileServiceClient(conni)

	return probe.runProbe(ctx, eclient, iclient)
}

func runProber(ctx context.Context) error {
	d := db.NewDatabase(ctx)
	probeState, err := d.LoadProberState(ctx)
	if err != nil {
		return err
	}

	first := ""
	fTime := int64(math.MaxInt64)
	for _, probe := range probers {
		lrt := time.Since(time.Unix(probeState.GetLastRun()[probe.getName()], 0))
		if lrt > probe.getFrequency() && lrt.Milliseconds() < fTime {
			first = probe.getName()
			fTime = lrt.Milliseconds()

		}
	}

	if first != "" {
		for _, probe := range probers {
			if probe.getName() == first {
				return runProbe(ctx, probe)
			}
		}
	}

	return nil
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()

	err := runProber(ctx)

	if err != nil {
		// Raise a ticket with this error
		gclient, err := ghb_client.GetClientInternal()
		if err != nil {
			log.Fatalf("%v", err)
		}

		gclient.CreateIssue(ctx, &ghpb.CreateIssueRequest{
			User:  "brotherlogic",
			Repo:  "gramophile",
			Title: "Prober Error",
			Body:  fmt.Sprintf("%v", err),
		})
	}
}
