package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

func GetSaleCandidate() *CLIModule {
	return &CLIModule{
		Command: "salecandidate",
		Help:    "Get sale candidate for an organisation",
		Execute: executeSaleCandidate,
	}
}

func formatSaleCandidate(rec *pb.Record) string {
	artist := getArtist(rec.GetRelease())
	title := rec.GetRelease().GetTitle()
	rating := rec.GetRelease().GetRating()
	packageScore := rec.GetPackageScore()

	var medianPrice float32
	if rec.GetMedianPrice() != nil {
		medianPrice = float32(rec.GetMedianPrice().GetValue()) / 100.0
	}

	var arrivalStr string
	if rec.GetArrived() > 0 {
		arrivalStr = time.Unix(0, rec.GetArrived()).Format("2006-01-02 15:04:05")
	} else if rec.GetRelease().GetDateAdded() > 0 {
		arrivalStr = time.Unix(0, rec.GetRelease().GetDateAdded()).Format("2006-01-02 15:04:05")
	} else {
		arrivalStr = "Unknown"
	}

	return fmt.Sprintf("Artist:         %s\nTitle:          %s\nDiscogs Rating: %d\nPackage Score:  %d\nMedian Price:   $%.2f\nArrival Date:   %s",
		artist, title, rating, packageScore, medianPrice, arrivalStr)
}

func runSaleCandidate(ctx context.Context, client pb.GramophileEServiceClient, orgName string, w io.Writer) error {
	if orgName == "" {
		return status.Errorf(codes.InvalidArgument, "org_name cannot be empty")
	}

	resp, err := client.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetSaleCandidate{
			GetSaleCandidate: &pb.GetSaleCandidate{
				OrgName: orgName,
			},
		},
	})
	if err != nil {
		return err
	}

	if len(resp.GetRecords()) == 0 || resp.GetRecords()[0].GetRecord() == nil {
		return status.Errorf(codes.NotFound, "no sale candidate found for org %v", orgName)
	}

	output := formatSaleCandidate(resp.GetRecords()[0].GetRecord())
	fmt.Fprintln(w, output)
	return nil
}

func executeSaleCandidate(ctx context.Context, args []string) error {
	if len(args) < 1 {
		return status.Errorf(codes.InvalidArgument, "usage: gram salecandidate <org_name>")
	}

	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewGramophileEServiceClient(conn)
	return runSaleCandidate(ctx, client, args[0], os.Stdout)
}
