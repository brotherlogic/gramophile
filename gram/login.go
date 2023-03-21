package main

import (
	"context"

	"github.com/dghubble/oauth1"
	"github.com/pkg/browser"
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

func getLoginURL(ctx context.Context) string {
	return "http://www.google.com"
}

func execute(ctx context.Context, args []string) error {
	return browser.OpenURL(getLoginURL(ctx))
}
