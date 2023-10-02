package integration

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	queuelogic "github.com/brotherlogic/gramophile/queuelogic"
	"github.com/brotherlogic/gramophile/server"
	rstore_client "github.com/brotherlogic/rstore/client"
	"google.golang.org/grpc/metadata"

	pb "github.com/brotherlogic/gramophile/proto"
)

func TestUserBuiltPostLogin(t *testing.T) {
	ctx := context.Background()

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	qc := queuelogic.GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)
	s := server.BuildServer(d, di, qc)

	_, err := s.GetURL(ctx, &pb.GetURLRequest{})
	if err != nil {
		t.Fatalf("Unable to get URL: %v", err)
	}

	// We need to pretend that discogs has responded here
	ats, err := d.LoadLogins(ctx)
	if err != nil {
		t.Fatalf("Unable to load logins")
	}
	for _, attempt := range ats.GetAttempts() {
		attempt.UserSecret = "madeupsecret"
		attempt.UserToken = "madeuptoken"
	}
	d.SaveLogins(ctx, ats)

	login, err := s.GetLogin(ctx, &pb.GetLoginRequest{Token: "madeuptoken"})
	if err != nil {
		t.Fatalf("Unable to get login: %v", err)
	}

	nctx := metadata.AppendToOutgoingContext(context.Background(), "auth-token", login.GetAuth().GetToken())
	user, err := s.GetUser(nctx, &pb.GetUserRequest{})

	if err != nil {
		t.Fatalf("Unable to get user: %v", err)
	}

	if user.GetUser().GetUser().GetUsername() != "madeupuser" {
		t.Errorf("Bad user return: %v", user)
	}

}
