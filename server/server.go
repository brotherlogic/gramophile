package server

import (
	"context"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	db "github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/queue"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type Server struct {
	d     db.Database
	Queue *queue.Queue
}

func NewServer(ctx context.Context, token, secret, callback string) *Server {
	d := db.NewDatabase(ctx)
	return &Server{
		d:     d,
		Queue: queue.GetQueue(background.GetBackgroundRunner(d, token, secret, callback), discogs.DiscogsWithAuth(token, secret, callback)),
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
			idt := md["auth-tokn"][0]

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

	return s.d.GetUser(ctx, key)
}
