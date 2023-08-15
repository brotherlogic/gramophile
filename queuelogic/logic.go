package queuelogic

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	rstore_client "github.com/brotherlogic/rstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

var (
	QUEUE_PREFIX = "gramophile/taskqueue/"

	queueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_qlen",
		Help: "The length of the working queue I think yes",
	})
	queueLast = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_last_proc",
		Help: "The length of the working queue I think yes",
	}, []string{"code"})
)

type queue struct {
	rstore rstore_client.RStoreClient
	b      *background.BackgroundRunner
	d      discogs.Discogs
	db     db.Database
}

func GetQueue(r rstore_client.RStoreClient, b *background.BackgroundRunner, d discogs.Discogs, db db.Database) *queue {
	return &queue{
		b: b, d: d, rstore: r, db: db,
	}
}

func (q *queue) Run() {
	log.Printf("Running queue with %+v", q.d)
	for {
		ctx := context.Background()
		entry, err := q.getNextEntry(ctx)
		if status.Code(err) != codes.NotFound {
			log.Printf("Got Entry: %v and %v", entry, err)
		}
		var erru error
		if err == nil {
			user, errv := q.db.GetUser(ctx, entry.GetAuth())
			err = errv
			erru = errv
			if err == nil {
				user.User.UserSecret = user.UserSecret
				user.User.UserToken = user.UserToken
				d := q.d.ForUser(user.GetUser())
				log.Printf("GOT USER: %+v and %+v", user, d)
				err = q.ExecuteInternal(ctx, d, entry)
				queueLast.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
			}
		}

		if status.Code(err) != codes.NotFound {
			log.Printf("Ran Entry: %v - %v", err, erru)
		}

		// Back off on any type of error - unless we failed to find the user (becuase they've been deleted)
		// Or because we've run an update on something that's not found
		if err == nil || status.Code(erru) == codes.NotFound || status.Code(err) == codes.NotFound {
			q.delete(ctx, entry)
		} else {
			if entry != nil {
				time.Sleep(time.Second * time.Duration(entry.GetBackoffInSeconds()))
			}
			time.Sleep(time.Minute)
		}
		time.Sleep(time.Second * 2)
	}
}

func (q *queue) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	var elems []*pb.QueueElement
	for _, key := range keys.GetKeys() {
		data, err := q.rstore.Read(ctx, &rspb.ReadRequest{Key: key})
		if err != nil {
			return nil, err
		}

		entry := &pb.QueueElement{}
		err = proto.Unmarshal(data.GetValue().GetValue(), entry)
		if err != nil {
			return nil, err
		}
		elems = append(elems, entry)
	}

	return &pb.ListResponse{Elements: elems}, nil
}

func (q *queue) Execute(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	user, err := q.db.GetUser(ctx, req.Element.GetAuth())
	if err != nil {
		return nil, err
	}
	d := q.d.ForUser(user.GetUser())
	return &pb.EnqueueResponse{}, q.ExecuteInternal(ctx, d, req.GetElement())
}

func (q *queue) ExecuteInternal(ctx context.Context, d discogs.Discogs, entry *pb.QueueElement) error {
	switch entry.Entry.(type) {
	case *pb.QueueElement_RefreshIntents:
		r, err := q.db.GetRecord(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId())
		if err != nil {
			return err
		}
		i, err := q.db.GetIntent(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId())
		if err != nil {
			return err
		}
		v := q.b.ProcessIntents(ctx, d, r, i, entry.GetAuth())
		log.Printf("Processed intent (%v) -> %v", i, v)
		return v
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth())
	case *pb.QueueElement_RefreshCollection:
		if entry.GetRefreshCollection().GetPage() == 1 {
			entry.GetRefreshCollection().RefreshId = time.Now().UnixNano()
		}

		rval, err := q.b.ProcessCollectionPage(ctx, d, entry.GetRefreshCollection().GetPage(), entry.GetRefreshCollection().GetRefreshId())
		log.Printf("Processed collection page: %v %v", rval, err)

		if err != nil {
			return err
		}
		if entry.GetRefreshCollection().GetPage() == 1 {
			for i := int32(2); i <= rval; i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().Unix() + int64(i),
					Entry: &pb.QueueElement_RefreshCollection{
						RefreshCollection: &pb.RefreshCollectionEntry{
							Page: i, RefreshId: entry.GetRefreshCollection().GetRefreshId()}},
					Auth: entry.GetAuth(),
				}})
				if err != nil {
					return err
				}
			}

			// If we've got here, update the user
			user, err := q.db.GetUser(ctx, entry.GetAuth())
			if err != nil {
				return err
			}
			user.LastCollectionRefresh = time.Now().Unix()
			return q.db.SaveUser(ctx, user)
		}
		return nil
	}

	return status.Errorf(codes.NotFound, "Unable to handle %v", entry)
}

func (q *queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, entry.GetRunDate())})
	return err
}

func (q *queue) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	log.Printf("Enqueue: %v", req)
	data, err := proto.Marshal(req.GetElement())
	if err != nil {
		return nil, err
	}
	_, err = q.rstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("%v%v", QUEUE_PREFIX, req.GetElement().GetRunDate()),
		Value: &anypb.Any{Value: data},
	})

	if err == nil {
		queueLen.Inc()
	}

	return &pb.EnqueueResponse{}, err
}

func (q *queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	queueLen.Set(float64(len(keys.GetKeys())))

	if len(keys.GetKeys()) == 0 {
		return nil, status.Errorf(codes.NotFound, "No queue entries")
	}

	data, err := q.rstore.Read(ctx, &rspb.ReadRequest{Key: keys.GetKeys()[0]})
	if err != nil {
		return nil, err
	}

	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	return entry, err
}
