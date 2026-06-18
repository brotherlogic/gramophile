package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	url, token, secret, err := s.di.GetLoginURL()
	if err != nil {
		return nil, fmt.Errorf("bad get for login ulr: %v", err)
	}

	attempts, err := s.d.LoadLogins(ctx)
	if err != nil {
		return nil, fmt.Errorf("bad load of logins: %v", err)
	}

	var newAttempts []*pb.UserLoginAttempt
	for _, attempt := range attempts.GetAttempts() {
		if time.Since(time.Unix(0, attempt.GetDateAdded())) < time.Minute*15 {
			newAttempts = append(newAttempts, attempt)
		}
	}
	attempts.Attempts = newAttempts

	attempts.Attempts = append(attempts.Attempts,
		&pb.UserLoginAttempt{
			RequestToken: token,
			Secret:       secret,
			DateAdded:    time.Now().UnixNano(),
		})

	log.Printf("Attempting this: %v", token)

	return &pb.GetURLResponse{URL: url, Token: token}, s.d.SaveLogins(ctx, attempts)
}

func (s *Server) GetLogin(ctx context.Context, req *pb.GetLoginRequest) (*pb.GetLoginResponse, error) {
	s.loginMutex.Lock()
	defer s.loginMutex.Unlock()

	attempts, err := s.d.LoadLogins(ctx)
	if err != nil {
		return nil, err
	}

	var newAttempts []*pb.UserLoginAttempt
	var foundAttempt *pb.UserLoginAttempt

	for _, attempt := range attempts.GetAttempts() {
		if time.Since(time.Unix(0, attempt.GetDateAdded())) < time.Minute*15 {
			if attempt.RequestToken == req.GetToken() && attempt.GetUserSecret() != "" {
				foundAttempt = attempt
			} else {
				newAttempts = append(newAttempts, attempt)
			}
		}
	}

	if foundAttempt != nil {
		user, err := s.d.GenerateToken(ctx, foundAttempt.GetUserToken(), foundAttempt.GetUserSecret())
		if err != nil {
			return nil, err
		}

		attempts.Attempts = newAttempts
		err = s.d.SaveLogins(ctx, attempts)
		if err != nil {
			return nil, err
		}

		// Enrich and store the user
		log.Printf("from %v got %v and %v", foundAttempt, user, err)
		sd := s.di.ForUser(&pbd.User{UserToken: foundAttempt.GetUserToken(), UserSecret: foundAttempt.GetUserSecret()})
			duser, err := sd.GetDiscogsUser(ctx)
			if err != nil {
				return nil, err
			}
			user.User = duser
			user.State = pb.StoredUser_USER_STATE_REFRESHING

			// Trigger a low-pri collection update
			s.qc.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 15,
					Priority:         pb.QueueElement_PRIORITY_LOW,
					Intention:        "New User Refresh",
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
					},
				},
			})

		return &pb.GetLoginResponse{Auth: user.GetAuth()}, s.d.SaveUser(ctx, user)
	}

	log.Printf("Tokens: %v", attempts)
	return nil, status.Errorf(codes.DataLoss, "unable to locate token in db")
}
