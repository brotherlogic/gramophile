package background

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	Intention = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_intention",
		Help: "The length of the working queue I think yes",
	}, []string{"intention"})
	MarkerCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gramophile_marker_rejects",
		Help: "The length of the working queue I think yes",
	})
	Rintention = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_refresh_intention",
		Help: "The length of the working queue I think yes",
	}, []string{"intention"})
)

type TaskHandler interface {
	Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error
	Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error
	GetDeduplicationKey(entry *pb.QueueElement) string
}

type BackgroundRunner struct {
	db                    db.Database
	key, secret, callback string
	ReleaseRefresh        int64
	handlers              map[string]TaskHandler
}

func GetBackgroundRunner(db db.Database, key, secret, callback string) *BackgroundRunner {
	br := &BackgroundRunner{
		db:       db,
		key:      key,
		secret:   secret,
		callback: callback,
		handlers: make(map[string]TaskHandler),
	}
	br.RegisterAllHandlers()
	return br
}

func (b *BackgroundRunner) RegisterTaskHandler(entryType string, handler TaskHandler) {
	b.handlers[entryType] = handler
}

func (b *BackgroundRunner) getHandler(entry *pb.QueueElement) (TaskHandler, error) {
	entryType := fmt.Sprintf("%T", entry.GetEntry())
	handler, ok := b.handlers[entryType]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "No handler registered for task type: %v", entryType)
	}
	return handler, nil
}

func (b *BackgroundRunner) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	handler, err := b.getHandler(entry)
	if err != nil {
		return err
	}
	return handler.Execute(ctx, d, u, entry, enqueue)
}

func (b *BackgroundRunner) Validate(ctx context.Context, entry *pb.QueueElement) error {
	handler, err := b.getHandler(entry)
	if err != nil {
		return err
	}
	return handler.Validate(ctx, b.db, entry)
}

func (b *BackgroundRunner) GetDeduplicationKey(entry *pb.QueueElement) string {
	handler, err := b.getHandler(entry)
	if err != nil {
		return ""
	}
	return handler.GetDeduplicationKey(entry)
}

func getRefKey(ctx context.Context) (string, error) {
	md, found := metadata.FromIncomingContext(ctx)
	if found {
		if _, ok := md["queue-key"]; ok {
			idt := md["queue-key"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	md, found = metadata.FromOutgoingContext(ctx)
	if found {
		if _, ok := md["queue-key"]; ok {
			idt := md["queue-key"][0]

			if idt != "" {
				return idt, nil
			}
		}
	}

	return "", status.Errorf(codes.NotFound, "Could not extract token from incoming or outgoing")
}

func qlog(ctx context.Context, str string, v ...any) {
	key, err := getRefKey(ctx)
	if err != nil {
		log.Printf("Unable to get ref key: %v", err)
		log.Printf(str, v...)
		return
	}

	prefix := fmt.Sprintf("%v: ", key)
	log.Printf(prefix+str, v...)
}
