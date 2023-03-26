package main

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"

	"github.com/dghubble/oauth1"
	"github.com/pkg/browser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var authenticateEndpoint = oauth1.Endpoint{
	RequestTokenURL: "https://api.discogs.com/oauth/request_token",
	AuthorizeURL:    "https://www.discogs.com/oauth/authorize",
	AccessTokenURL:  "https://api.discogs.com/oauth/access_token",
}

func GetLogin() *CLIModule {
	return &CLIModule{
		Command: "login",
		Help:    "Logs into the Discogs system",
		Execute: execute,
	}
}

func getLoginURL(ctx context.Context) (string, error) {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", err
	}

	client := pb.NewGramophileEServiceClient(conn)
	resp, err := client.GetURL(ctx, &pb.GetURLRequest{})
	if err != nil {
		return "", err
	}

	return resp.GetURL(), nil
}

func execute(ctx context.Context, args []string) error {
	val, err := getLoginURL(ctx)
	if err != nil {
		return err
	}
	return browser.OpenURL(val)
}
