package server

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/brotherlogic/discogs"
	db "github.com/brotherlogic/gramophile/db"

	pbgd "github.com/brotherlogic/godiscogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	pbrc "github.com/brotherlogic/recordcollection/proto"

	"google.golang.org/grpc"
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

func generateContext(ctx context.Context, origin string) context.Context {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	hostname := "kubernetes"
	tracev := fmt.Sprintf("%v-%v-%v-%v", origin, time.Now().Unix(), r.Int63(), hostname)
	mContext := metadata.AppendToOutgoingContext(ctx, "trace-id", tracev)
	return mContext
}

func (s *Server) updateRecord(ctx context.Context, id int32) error {
	conn, err := grpc.Dial("argon:57724", grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pbrc.NewRecordCollectionServiceClient(conn)
	nctx := generateContext(ctx, "gramophile")
	_, err = client.UpdateRecord(nctx, &pbrc.UpdateRecordRequest{
		Reason: "ping_from_gramophile",
		Update: &pbrc.Record{
			Release:  &pbgd.Release{InstanceId: id},
			Metadata: &pbrc.ReleaseMetadata{NeedsGramUpdate: true},
		},
	})
	return err
}
