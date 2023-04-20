package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	rstore_client "github.com/brotherlogic/rstore/client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

var (
	QUEUE_SUFFIX = "gramophile/taskqueue/"

	queueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_qlen",
		Help: "The length of the working queue I think yes",
	})

	internalPort = flag.Int("internal_port", 8080, "Port to serve internal grpc traffic")
	metricsPort  = flag.Int("metrics_port", 8081, "Metrics port")
)

type Queue struct {
	rstore *rstore_client.RStoreClient
	b      *background.BackgroundRunner
	d      discogs.Discogs
	db     db.DB
}

func GetQueue(b *background.BackgroundRunner, d discogs.Discogs) *Queue {
	return &Queue{
		b: b, d: d,
	}
}

func (q *Queue) run() {
	for {
		ctx := context.Background()
		entry, err := q.getNextEntry(ctx)
		if err == nil {
			user, err := q.db.GetUser(ctx, entry.GetAuth())
			if err == nil {
				d := q.d.ForUser(user.GetUser())
				err = q.ExecuteInternal(ctx, d, entry)
			}

		}

		// Back off on any type of error
		if err == nil {
			q.delete(ctx, entry)
		} else {
			if entry != nil {
				time.Sleep(time.Second * time.Duration(entry.GetBackoffInSeconds()))
			} else {
				time.Sleep(time.Second * 10)
			}
		}

		time.Sleep(time.Hour)
	}
}

func (q *Queue) Execute(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	user, err := q.db.GetUser(ctx, req.Element.GetAuth())
	if err != nil {
		return nil, err
	}
	d := q.d.ForUser(user.GetUser())
	return &pb.EnqueueResponse{}, q.ExecuteInternal(ctx, d, req.GetElement())
}

func (q *Queue) ExecuteInternal(ctx context.Context, d discogs.Discogs, entry *pb.QueueElement) error {
	switch entry.Entry.(type) {
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth())
	case *pb.QueueElement_RefreshCollection:
		rval, err := q.b.ProcessCollectionPage(ctx, d, entry.GetRefreshCollection().GetPage())
		if err != nil {
			return err
		}
		if entry.GetRefreshCollection().GetPage() == 1 {
			for i := int32(2); i <= rval; i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().Unix() + int64(i),
					Entry:   &pb.QueueElement_RefreshCollection{RefreshCollection: &pb.RefreshCollectionEntry{Page: i}},
					Auth:    entry.GetAuth(),
				}})
				if err != nil {
					return err
				}
			}
		}
		return nil
	}

	return status.Errorf(codes.NotFound, "Unable to handle %v", entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v/%v", QUEUE_SUFFIX, entry.GetRunDate())})
	return err
}

func (q *Queue) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	data, err := proto.Marshal(req.GetElement())
	if err != nil {
		return nil, err
	}
	_, err = q.rstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("%v/%v", req.GetElement().GetRunDate()),
		Value: &anypb.Any{Value: data},
	})
	return &pb.EnqueueResponse{}, err
}

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_SUFFIX})
	if err != nil {
		return nil, err
	}

	queueLen.Set(float64(len(keys.GetKeys())))

	data, err := q.rstore.Read(ctx, &rspb.ReadRequest{Key: keys.GetKeys()[0]})
	if err != nil {
		return nil, err
	}

	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	return entry, err
}

func main() {
	rstorec, err := rstore_client.GetClient()
	if err != nil {
		log.Fatalf("Unable to connect to rstore: %v", err)
	}
	queue := &Queue{
		rstore: rstorec,
	}

	lis, err2 := net.Listen("tcp", fmt.Sprintf(":%d", *internalPort))
	if err2 != nil {
		log.Fatalf("gramophile is unable to listen on the internal grpc port %v: %v", *internalPort, err)
	}
	gsInternal := grpc.NewServer()
	go func() {
		if err := gsInternal.Serve(lis); err != nil {
			log.Fatalf("queue is unable to serve internal grpc: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
		log.Fatalf("gramophile is unable to serve metrics: %v", err)
	}()

	pb.RegisterQueueServiceServer(gsInternal, queue)

	queue.run()
}
