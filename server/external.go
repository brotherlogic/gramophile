package server

import (
	"context"
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

	attempts, err := s.d.loadLogins(ctx)
	if err != nil {
		return nil, err
	}
	attempts.Attempts = append(attempts.Attempts,
		&pb.UserLoginAttempt{
			RequestToken: token,
			Secret:       secret,
			DateAdded:    time.Now().Unix(),
		})

	return &pb.GetURLResponse{URL: url, Token: token}, s.d.saveLogins(ctx, attempts)
}

func (s *Server) GetLogin(ctx context.Context, req *pb.GetLoginRequest) (*pb.GetLoginResponse, error) {
	attempts, err := s.d.loadLogins(ctx)
	if err != nil {
		return nil, err
	}

	for _, attempt := range attempts.GetAttempts() {
		if attempt.RequestToken == req.GetToken() {
			token, err := s.d.generateToken(ctx, attempt.GetUserToken(), attempt.GetUserSecret())
			if err != nil {
				return nil, err
			}
			return &pb.GetLoginResponse{Auth: token}, nil
		}
	}

	return nil, status.Errorf(codes.DataLoss, "Unable to locate token in db")
}
