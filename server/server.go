package server

import (
	"context"
	"os"

	"github.com/brotherlogic/discogs"
	db "github.com/brotherlogic/gramophile/db"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	d  db.Database
	di discogs.Discogs
}

func NewServer(ctx context.Context, token, secret, callback string) *Server {
	d := db.NewDatabase(ctx)
	di := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))

	return &Server{
		d:  d,
		di: di,
	}
}

func GetContextKey(ctx context.Context) (string, error) {
	md, found := metadata.FromIncomingContext(ctx)
	if found {
		if _, ok := md["auth-token"]; ok {
			idt := md["auth-token"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	md, found = metadata.FromOutgoingContext(ctx)
	if found {
		if _, ok := md["auth-token"]; ok {
			idt := md["auth-token"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	return "", status.Errorf(codes.NotFound, "Could not extract token from incoming or outgoing")
}

func (s *Server) getUser(ctx context.Context) (*pb.StoredUser, error) {
	key, err := GetContextKey(ctx)
	if err != nil {
		return nil, err
	}

	user, err := s.d.GetUser(ctx, key)

	return user, err
}
