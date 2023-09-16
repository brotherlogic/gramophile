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

type Queue struct {
	rstore rstore_client.RStoreClient
	b      *background.BackgroundRunner
	d      discogs.Discogs
	db     db.Database
}

func GetQueue(r rstore_client.RStoreClient, b *background.BackgroundRunner, d discogs.Discogs, db db.Database) *Queue {
	log.Printf("GETTING QUEUE")
	return &Queue{
		b: b, d: d, rstore: r, db: db,
	}
}

func (q *Queue) FlushQueue(ctx context.Context) {
	elem, err := q.getNextEntry(ctx)
	log.Printf("First Entry: %v", elem)

	for err == nil {
		user, errv := q.db.GetUser(ctx, elem.GetAuth())
		if errv != nil {
			log.Fatalf("unable to get user to flush queue: %v -> %v", errv, elem.GetAuth())
		}
		user.User.UserSecret = user.UserSecret
		user.User.UserToken = user.UserToken
		d := q.d.ForUser(user.GetUser())
		errp := q.ExecuteInternal(ctx, d, user, elem)
		if errp == nil {
			q.delete(ctx, elem)
		} else {
			log.Fatalf("Failed to execute internal: %v", errp)
		}

		elem, err = q.getNextEntry(ctx)
	}
}

func (q *Queue) Run() {
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
				err = q.ExecuteInternal(ctx, d, user, entry)
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

func (q *Queue) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
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

func (q *Queue) Execute(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	user, err := q.db.GetUser(ctx, req.Element.GetAuth())
	if err != nil {
		return nil, err
	}
	d := q.d.ForUser(user.GetUser())
	return &pb.EnqueueResponse{}, q.ExecuteInternal(ctx, d, user, req.GetElement())
}

func (q *Queue) ExecuteInternal(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement) error {
	log.Printf("Running queue entry: %v -> %v", entry, u)
	switch entry.Entry.(type) {
	case *pb.QueueElement_MoveRecord:
		rec, err := q.db.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), entry.GetMoveRecord().GetRecordIid())
		if err != nil {
			return fmt.Errorf("unable to get record: %w", err)
		}

		fNum := int64(-1)
		for _, folder := range u.GetFolders() {
			if folder.GetName() == entry.GetMoveRecord().GetMoveFolder() {
				fNum = int64(folder.GetId())
			}
		}

		if fNum < 0 {
			return status.Errorf(codes.NotFound, "Folder %v was not found", entry.GetMoveRecord().GetMoveFolder())
		}

		err = d.SetFolder(ctx, rec.GetRelease().GetInstanceId(), rec.GetRelease().GetId(), int64(rec.GetRelease().GetFolderId()), fNum)
		if err != nil {
			return fmt.Errorf("unable to move record: %w", err)
		}

		log.Printf("Setting folder: %v", fNum)

		//Update and save record
		rec.GetRelease().FolderId = int32(fNum)
		return q.db.SaveRecord(ctx, u.GetUser().GetDiscogsUserId(), rec)

	case *pb.QueueElement_MoveRecords:
		return q.b.RunMoves(ctx, u, q.Enqueue)
	case *pb.QueueElement_UpdateSale:
		return q.b.UpdateSalePrice(ctx, d, entry.GetUpdateSale().GetSaleId(), entry.GetUpdateSale().GetNewPrice())
	case *pb.QueueElement_RefreshWants:
		return q.b.RefereshWants(ctx, d)
	case *pb.QueueElement_RefreshWantlists:
		return q.b.RefreshWantlists(ctx, d, entry.GetAuth())
	case *pb.QueueElement_LinkSales:
		err := q.b.LinkSales(ctx, u)
		if err != nil {
			return fmt.Errorf("unable to link sales: %w", err)
		}
		return nil
	case *pb.QueueElement_RefreshSales:
		if entry.GetRefreshSales().GetPage() == 1 {
			entry.GetRefreshSales().RefreshId = time.Now().UnixNano()
		}
		pages, err := q.b.SyncSales(ctx, d, entry.GetRefreshSales().GetPage(), entry.GetRefreshSales().GetRefreshId())

		if err != nil {
			return err
		}

		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return fmt.Errorf("unable to get user: %w", err)
		}
		log.Printf("Got user: %v", user)

		if entry.GetRefreshSales().GetPage() == 1 {
			for i := int32(2); i <= pages.GetPages(); i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().Unix() + int64(i),
					Entry: &pb.QueueElement_RefreshSales{
						RefreshSales: &pb.RefreshSales{
							Page: i, RefreshId: entry.GetRefreshCollection().GetRefreshId()}},
					Auth: entry.GetAuth(),
				}})
				if err != nil {
					return fmt.Errorf("unable to enqueue: %w", err)
				}
			}

			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
				RunDate: time.Now().Unix() + int64(pages.GetPages()) + 10,
				Entry: &pb.QueueElement_LinkSales{
					LinkSales: &pb.LinkSales{
						RefreshId: entry.GetRefreshCollection().GetRefreshId()}},
				Auth: entry.GetAuth(),
			}})
			if err != nil {
				return fmt.Errorf("dunable to enqueue link job: %v", err)
			}

			// If we've got here, update the user
			user.LastSaleRefresh = time.Now().Unix()
			err = q.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to sell user: %w", err)
			}
			return err
		}

		// Adjust all sale prices
		return q.b.AdjustSales(ctx, user.GetConfig().GetSaleConfig(), user, q.Enqueue)
	case *pb.QueueElement_AddFolderUpdate:
		err := q.b.AddFolder(ctx, entry.GetAddFolderUpdate().GetFolderName(), d, u)
		if err != nil {
			return fmt.Errorf("unable to create folder: %w", err)
		}
		return nil
	case *pb.QueueElement_RefreshIntents:
		r, err := q.db.GetRecord(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId())
		if err != nil {
			return fmt.Errorf("unable to get record from %v: %w", d.GetUserId(), err)
		}
		i, err := q.db.GetIntent(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId())
		if err != nil {
			return fmt.Errorf("unable to get intent: %w", err)
		}
		v := q.b.ProcessIntents(ctx, d, r, i, entry.GetAuth())
		log.Printf("Processed intent (%v) -> %v", i, v)
		return v
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth())
	case *pb.QueueElement_RefreshUpdates:
		return q.b.RefreshUpdates(ctx, d)
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
					return fmt.Errorf("unable to enqueue: %w", err)
				}
			}

			// If we've got here, update the user
			user, err := q.db.GetUser(ctx, entry.GetAuth())
			if err != nil {
				return fmt.Errorf("unable to get user: %w", err)
			}
			user.LastCollectionRefresh = time.Now().Unix()
			err = q.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to sell user: %w", err)
			}
			return err
		}
		return nil
	}

	return status.Errorf(codes.NotFound, "Unable to this handle %v -> %v", entry, entry.Entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, entry.GetRunDate())})
	return err
}

func (q *Queue) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
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

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
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
