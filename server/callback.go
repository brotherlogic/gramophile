package server

import (
	"net/http"
	"os"

	"github.com/brotherlogic/discogs"
)

func (s *Server) handleCallback(w http.ResponseWriter, r *http.Request) {
	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))

	token := r.URL.Query().Get("oauth_token")
	secret := r.URL.Query().Get("oauth_secret")

	d.HandleDiscogsResponse(ctx, secret, token, verifier)
}
