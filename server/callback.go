package server

import (
	"log"
	"net/http"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	token := r.URL.Query().Get("oauth_token")
	verifier := r.URL.Query().Get("oauth_verifier")

	s.loginMutex.Lock()
	logins, err := s.d.LoadLogins(ctx)
	if err != nil {
		s.loginMutex.Unlock()
		log.Printf("Bad load logins: %v", err)
		http.Error(w, "Failed to load logins", http.StatusInternalServerError)
		return
	}
	s.loginMutex.Unlock()

	var matchingLogin *pb.UserLoginAttempt
	for _, login := range logins.GetAttempts() {
		if login.RequestToken == token {
			matchingLogin = login
			break
		}
	}

	if matchingLogin == nil {
		log.Printf("Unable to locate user in login attempts")
		http.Error(w, "Unable to locate user in login attempts", http.StatusBadRequest)
		return
	}

	tokenResponse, secret, err := s.di.HandleDiscogsResponse(ctx, matchingLogin.GetSecret(), token, verifier)
	if err != nil {
		log.Printf("Failed to handle Discogs response: %v", err)
		http.Error(w, "Failed to handle Discogs response", http.StatusInternalServerError)
		return
	}

	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	// Reload logins to avoid concurrent write races
	logins, err = s.d.LoadLogins(ctx)
	if err != nil {
		log.Printf("Bad load logins on update: %v", err)
		http.Error(w, "Failed to load logins", http.StatusInternalServerError)
		return
	}

	found := false
	for _, login := range logins.GetAttempts() {
		if login.RequestToken == token {
			login.UserSecret = secret
			login.UserToken = tokenResponse
			found = true
			break
		}
	}

	if !found {
		log.Printf("Unable to locate user in login attempts after callback")
		http.Error(w, "Unable to locate user in login attempts", http.StatusInternalServerError)
		return
	}

	log.Printf("Saving login: %v", token)
	err = s.d.SaveLogins(ctx, logins)
	if err != nil {
		log.Printf("Bad save logins: %v", err)
		http.Error(w, "Failed to save logins", http.StatusInternalServerError)
		return
	}
}
