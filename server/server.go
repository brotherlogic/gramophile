package server

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	queue_client "github.com/brotherlogic/gramophile/queue_client"

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
	qc queue_client.QueueClient
}

func BuildServer(d db.Database, di discogs.Discogs, qc queue_client.QueueClient) *Server {
	return &Server{
		d:  d,
		di: di,
		qc: qc,
	}
}

func NewServer(ctx context.Context, token, secret, callback string) *Server {
	d := db.NewDatabase(ctx)
	di := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))
	qc, err := queue_client.GetClient()
	if err != nil {
		log.Fatalf("unable to reach queue: %v", err)
	}

	return &Server{
		d:  d,
		di: di,
		qc: qc,
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

	if user.GetUser().GetUserSecret() == "" {
		user.GetUser().UserSecret = user.GetUserSecret()
		user.GetUser().UserToken = user.GetUserToken()
	}

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
	defer conn.Close()

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
