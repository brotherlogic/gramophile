package server

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/brotherlogic/discogs"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))

	ctx := context.Background()

	token := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	logins, err := s.d.loadLogins(ctx)
	if err != nil {
		log.Fatalf("Bad: %v", err)
	}

	for _, login := range logins.GetAttempts() {
		if login.RequestToken == token {
			user, err := d.HandleDiscogsResponse(ctx, login.GetSecret(), token, verifier)
			log.Printf("Boing:%v ->  %v and %v from %v", token, user, err, verifier)
		}
	}
}
