package server

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))
	url, token, secret, err := d.GetLoginURL()
	if err != nil {
		return nil, fmt.Errorf("bad get for login ulr: %v", err)
	}

	attempts, err := s.d.LoadLogins(ctx)
	if err != nil {
		return nil, fmt.Errorf("bad load of logins: %v", err)
	}
	attempts.Attempts = append(attempts.Attempts,
		&pb.UserLoginAttempt{
			RequestToken: token,
			Secret:       secret,
			DateAdded:    time.Now().Unix(),
		})

	log.Printf("Attempting: %v", token)

	return &pb.GetURLResponse{URL: url, Token: token}, s.d.SaveLogins(ctx, attempts)
}

func (s *Server) GetLogin(ctx context.Context, req *pb.GetLoginRequest) (*pb.GetLoginResponse, error) {
	attempts, err := s.d.LoadLogins(ctx)
	if err != nil {
		return nil, err
	}

	for _, attempt := range attempts.GetAttempts() {
		if attempt.RequestToken == req.GetToken() {
			token, err := s.d.GenerateToken(ctx, attempt.GetUserToken(), attempt.GetUserSecret())
			if err != nil {
				return nil, err
			}
			return &pb.GetLoginResponse{Auth: token}, nil
		}
	}

	return nil, status.Errorf(codes.DataLoss, "Unable to locate token in db")
}
