package main

import (
	"context"
	"os"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/golang/protobuf/proto"

	"github.com/dghubble/oauth1"
	"github.com/pkg/browser"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
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

func getLoginURL(ctx context.Context) (string, string, error) {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return "", "", err
	}

	client := pb.NewGramophileEServiceClient(conn)
	resp, err := client.GetURL(ctx, &pb.GetURLRequest{})
	if err != nil {
		return "", "", err
	}

	return resp.GetURL(), resp.GetToken(), nil
}

func getAuthToken(ctx context.Context, token string) (*pb.GramophileAuth, error) {
	conn, err := grpc.Dial("gramophile-grpc.brotherlogic-backend.com:80", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewGramophileEServiceClient(conn)
	resp, err := client.GetLogin(ctx, &pb.GetLoginRequest{Token: token})
	if err != nil {
		return nil, err
	}

	return resp.GetAuth(), nil
}

func execute(ctx context.Context, args []string) error {
	val, token, err := getLoginURL(ctx)
	if err != nil {
		return err
	}

	err = browser.OpenURL(val)
	if err != nil {
		return err
	}

	// Only try to retreive the login details for 5 minutes
	t := time.Now()
	for time.Since(t) < time.Minute*5 {
		time.Sleep(time.Second * 5)

		auth, err := getAuthToken(ctx, token)
		if err != nil {
			if status.Code(err) != codes.FailedPrecondition {
				return err
			}
		} else {
			break
		}

		f, err := os.OpenFile("~/.gramophile", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer f.Close()
		return proto.MarshalText(f, auth)
	}

	return status.Errorf(codes.DeadlineExceeded, "Unable to get login token after 5 minutes")
}
