package queuelogic

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"sort"
	"strconv"
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
	scraper_client "github.com/brotherlogic/scraper/client"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

var (
	QUEUE_PREFIX    = "gramophile/taskqueue/"
	DL_QUEUE_PREFIX = "gramophile/dlq/"

	queueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_qlen",
		Help: "The length of the working queue I think yes",
	})
	dlQeueLen = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_dlqlen",
		Help: "The length of the working queue I think yes",
	})

	queueLast = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_last_proc",
		Help: "The length of the working queue I think yes",
	}, []string{"code", "type"})
	queueRun = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_proc",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
	queueSleep = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_queue_sleep",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
	queueRunTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gramophile_queue_time",
		Help:    "The length of the working queue I think yes",
		Buckets: []float64{1000, 2000, 4000, 8000, 16000, 32000, 64000, 128000, 256000, 512000, 1024000, 2048000, 4096000},
	}, []string{"type"})
	queueLoadTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gramophile_queue_load_time",
		Help:    "The length of the working queue I think yes",
		Buckets: []float64{1000, 2000, 4000, 8000, 16000, 32000, 64000, 128000, 256000, 512000, 1024000, 2048000, 4096000},
	}, []string{"type"})
	queueAdd = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_adds",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
)

type Queue struct {
	rstore rstore_client.RStoreClient
	b      *background.BackgroundRunner
	d      discogs.Discogs
	db     db.Database
	keys   []int64
}

func GetQueue(r rstore_client.RStoreClient, b *background.BackgroundRunner, d discogs.Discogs, db db.Database) *Queue {
	log.Printf("GETTING QUEUE")
	sc, err := scraper_client.GetClient()
	if err != nil {
		panic(err)
	}
	d.SetDownloader(&DownloaderBridge{scraper: sc})

	log.Printf("Loading cache")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	keys, err := r.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		log.Fatalf("Unable to get keys: %v", err)
	}
	var ckeys []int64
	for _, key := range keys.GetKeys() {
		value, err := strconv.ParseInt(key[len(QUEUE_PREFIX):], 10, 64)
		if err != nil {
			log.Fatalf("Bad parse: %v (%v)", err, key)
		}
		ckeys = append(ckeys, value)
	}

	return &Queue{
		b: b, d: d, rstore: r, db: db, keys: ckeys,
	}
}

func (q *Queue) getRefreshMarker(ctx context.Context, user string, id int64) (int64, error) {
	entry, err := q.rstore.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id)})

	if err != nil {
		return -1, err
	}

	return int64(binary.BigEndian.Uint64(entry.GetValue().GetValue())), nil
}

func (q *Queue) getRefreshDateMarker(ctx context.Context, user string, id int64) (int64, error) {
	entry, err := q.rstore.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v-%v", user, id)})

	if err != nil {
		return -1, err
	}

	return int64(binary.BigEndian.Uint64(entry.GetValue().GetValue())), nil
}

func (q *Queue) setRefreshMarker(ctx context.Context, user string, id int64) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	_, err := q.rstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id),
		Value: &anypb.Any{Value: b},
	})

	if err != nil {
		return err
	}

	return nil
}

func (q *Queue) setRefreshDateMarker(ctx context.Context, user string, id int64) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	_, err := q.rstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v-%v", user, id),
		Value: &anypb.Any{Value: b},
	})

	if err != nil {
		return err
	}

	return nil
}

func (q *Queue) deleteRefreshMarker(ctx context.Context, user string, id int64) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id),
	})

	return err
}

func (q *Queue) deleteRefreshDateMarker(ctx context.Context, user string, id int64) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v-%v", user, id),
	})

	return err
}

func (q *Queue) FlushQueue(ctx context.Context) error {
	log.Printf("Flushing queue")
	elem, err := q.getNextEntry(ctx)
	log.Printf("First Entry: %v", elem)

	for err == nil {
		user, errv := q.db.GetUser(ctx, elem.GetAuth())
		if errv != nil {
			log.Fatalf("unable to get user to flush queue: %v -> %v", errv, elem.GetAuth())
		}
		user.User.UserSecret = user.UserSecret
		user.User.UserToken = user.UserToken
		log.Printf("USER: %v", user)
		d := q.d.ForUser(user.GetUser())
		errp := q.ExecuteInternal(ctx, d, user, elem)
		if errp == nil {
			q.delete(ctx, elem)
		} else {
			log.Printf("Failed to execute internal: %v -> %v", errp, elem)
			return errp
		}

		elem, err = q.getNextEntry(ctx)
		log.Printf("Post flush: %v", err)
	}

	return nil
}

func (q *Queue) Run() {
	log.Printf("Running queue with %+v", q.d)
	for {
		ctx := context.Background()
		t1 := time.Now()
		entry, err := q.getNextEntry(ctx)
		if status.Code(err) != codes.NotFound {
			log.Printf("Got Entry: %v and %v (%v)", entry, err, time.Since(t1))
		}
		var erru error
		if err == nil {
			user, errv := q.db.GetUser(ctx, entry.GetAuth())
			err = errv
			erru = errv
			if err == nil {
				if user.GetUser() == nil {
					user.User = &pbd.User{UserSecret: user.GetUserSecret(), UserToken: user.GetUserToken()}
				} else {
					user.GetUser().UserSecret = user.GetUserSecret()
					user.GetUser().UserToken = user.GetUserToken()
				}
				d := q.d.ForUser(user.GetUser())
				st := time.Now()
				err = q.ExecuteInternal(ctx, d, user, entry)
				log.Printf("Queue entry end %v in %v -> %v ", entry, time.Since(st), err)
				queueRunTime.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Observe(float64(time.Since(st).Milliseconds()))
				queueLast.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry()), "code": fmt.Sprintf("%v", status.Code(err))}).Inc()
			}
		}

		if status.Code(err) != codes.NotFound {
			log.Printf("Ran Entry: (%v) %v - %v", entry, err, erru)
		}

		// Back off on any type of error - unless we failed to find the user (becuase they've been deleted)
		// Or because we've run an update on something that's not found
		if err == nil || status.Code(erru) == codes.NotFound || status.Code(err) == codes.NotFound {
			q.delete(ctx, entry)
		} else {
			// This is discogs throttling us
			if status.Code(err) == codes.ResourceExhausted {
				log.Printf("Waiting for a minute to let our tokens regenerate")
				time.Sleep(time.Minute)
			} else {
				// Move this over to the DLQ
				data, err := proto.Marshal(entry)
				if err == nil {
					_, err = q.rstore.Write(ctx, &rspb.WriteRequest{
						Key:   fmt.Sprintf("%v%v", DL_QUEUE_PREFIX, entry.GetRunDate()),
						Value: &anypb.Any{Value: data},
					})
					dlQeueLen.Inc()

					if err == nil {
						q.delete(ctx, entry)
					}
				}
			}
		}
	}
}

func (q *Queue) Drain(ctx context.Context, req *pb.DrainRequest) (*pb.DrainResponse, error) {
	keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	for _, key := range keys.GetKeys() {
		_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: key})
		if err != nil {
			return nil, err
		}
	}

	return &pb.DrainResponse{Count: int32(len(keys.GetKeys()))}, nil
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
	log.Printf("Queue entry start: %v", entry)
	queueRun.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.Entry)}).Inc()
	if fmt.Sprintf("%T", entry.Entry) != "*proto.QueueElement_RefreshCollectionEntry" &&
		fmt.Sprintf("%T", entry.Entry) != "*proto.QueueElement_RefreshIntents" &&
		fmt.Sprintf("%T", entry.Entry) != "*proto.QueueElement_RefreshWants" &&
		fmt.Sprintf("%T", entry.Entry) != "*proto.QueueElement_RefreshWant" &&
		fmt.Sprintf("%T", entry.Entry) != "*proto.QueueElement_SyncWants" {
		log.Printf("Skipping '%T'", entry.Entry)
		return nil
	}
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
			return status.Errorf(codes.NotFound, "folder %v was not found", entry.GetMoveRecord().GetMoveFolder())
		}

		err = d.SetFolder(ctx, rec.GetRelease().GetInstanceId(), rec.GetRelease().GetId(), int64(rec.GetRelease().GetFolderId()), fNum)
		if err != nil {
			return fmt.Errorf("unable to move record: %w", err)
		}

		log.Printf("Setting folder: %v", fNum)

		//Update and save record
		rec.GetRelease().FolderId = int32(fNum)
		err = q.db.SaveRecordWithUpdate(ctx, u.GetUser().GetDiscogsUserId(), rec, &pb.RecordUpdate{
			Date:        time.Now().UnixNano(),
			Explanation: []string{fmt.Sprintf("Moved to %v following rule %v", entry.GetMoveRecord().GetMoveFolder(), entry.GetMoveRecord().GetRule())},
		})
		if err != nil {
			return err
		}

		_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_MoveRecords{
					MoveRecords: &pb.MoveRecords{},
				},
				Auth: entry.GetAuth(),
			}})
		return err
	case *pb.QueueElement_MoveRecords:
		return q.b.RunMoves(ctx, u, q.Enqueue)
	case *pb.QueueElement_AddMasterWant:
		return q.b.AddMasterWant(ctx, d, entry.GetAddMasterWant().GetWant())
	case *pb.QueueElement_UpdateSale:
		//Short cut if sale data is not complete
		if entry.GetUpdateSale().GetCondition() == "" {
			log.Printf("Skipping %v", entry)
			return nil
		}
		err := q.b.UpdateSalePrice(ctx, d, entry.GetUpdateSale().GetSaleId(), entry.GetUpdateSale().GetReleaseId(), entry.GetUpdateSale().GetCondition(), entry.GetUpdateSale().GetNewPrice(), entry.GetUpdateSale().GetMotivation())
		log.Printf("Updated sale price for %v -> %v", entry.GetUpdateSale().GetSaleId(), err)

		// Not Found means the sale was deleted - if so remove from the db
		if status.Code(err) == codes.NotFound {
			log.Printf("Deleting sale for %v (%v) since we can't locate the sale", entry.GetUpdateSale().GetReleaseId(), entry.GetUpdateSale().GetSaleId())
			return q.db.DeleteSale(ctx, u.GetUser().GetDiscogsUserId(), entry.GetUpdateSale().GetSaleId())
		}
		return err
	case *pb.QueueElement_RefreshWants:
		return q.b.RefreshWants(ctx, d)
	case *pb.QueueElement_RefreshWant:
		return q.b.RefreshWant(ctx, d, entry.GetRefreshWant().GetWant(), entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_SyncWants:
		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return fmt.Errorf("unable to get user: %w", err)
		}

		// Only refresh every 24 hours
		if time.Since(time.Unix(0, user.GetLastWantRefresh())) < time.Hour*24 {
			return nil
		}

		if entry.GetSyncWants().GetPage() == 1 {
			entry.GetSyncWants().RefreshId = time.Now().UnixNano()
		}
		pages, err := q.b.PullWants(ctx, d, entry.GetSyncWants().GetPage(), entry.GetSyncWants().GetRefreshId(), user.GetConfig().GetWantsConfig())
		if err != nil {
			return err
		}
		if entry.GetSyncWants().GetPage() == 1 {
			for i := int32(2); i <= pages; i++ {
				q.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate: time.Now().UnixNano() + int64(i),
						Entry: &pb.QueueElement_SyncWants{
							SyncWants: &pb.SyncWants{Page: i, RefreshId: entry.GetSyncWants().GetRefreshId()},
						},
						Auth: entry.GetAuth(),
					},
				})
			}
		}

		// If this is the final sync, let's run the alignment
		if entry.GetSyncWants().GetPage() >= pages {
			err = q.b.SyncWants(ctx, d, user, q.Enqueue)
			if err != nil {
				return err
			}

			err = q.b.AlignWants(ctx, d, user.GetConfig().GetWantsConfig())
			if err != nil {
				return err
			}

			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 60,
					Entry: &pb.QueueElement_RefreshWants{
						RefreshWants: &pb.RefreshWants{},
					},
				},
			})

			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 60,
					Entry: &pb.QueueElement_RefreshWantlists{
						RefreshWantlists: &pb.RefreshWantlists{},
					},
				},
			})

			user.LastWantRefresh = time.Now().UnixNano()
			return q.db.SaveUser(ctx, user)
		}

		return nil
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
			log.Printf("Starting Updating run for 2836578592 -> %v", entry.GetRefreshSales().GetRefreshId())
		}
		pages, err := q.b.SyncSales(ctx, d, entry.GetRefreshSales().GetPage(), entry.GetRefreshSales().GetRefreshId())

		if err != nil {
			return err
		}

		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return fmt.Errorf("unable to get user: %w", err)
		}
		log.Printf("Got user: %v with %v", user, entry.GetRefreshSales())

		if entry.GetRefreshSales().GetPage() == 1 {
			for i := int32(2); i <= pages.GetPages(); i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano() + int64(i),
					Entry: &pb.QueueElement_RefreshSales{
						RefreshSales: &pb.RefreshSales{
							Page: i, RefreshId: entry.GetRefreshSales().GetRefreshId()}},
					Auth: entry.GetAuth(),
				}})
				if err != nil {
					return fmt.Errorf("unable to enqueue: %w", err)
				}
			}

			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano() + int64(pages.GetPages()) + 10,
				Entry: &pb.QueueElement_LinkSales{
					LinkSales: &pb.LinkSales{
						RefreshId: entry.GetRefreshSales().GetRefreshId()}},
				Auth: entry.GetAuth(),
			}})
			if err != nil {
				return fmt.Errorf("dunable to enqueue link job: %v", err)
			}

			// If we've got here, update the user
			user.LastSaleRefresh = time.Now().UnixNano()
			err = q.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to sell user: %w", err)
			}
		}

		log.Printf("Checking for Clean %v vs %v", entry.GetRefreshSales().GetPage(), pages.GetPages())
		if entry.GetRefreshSales().GetPage() >= pages.GetPages() {
			err := q.b.CleanSales(ctx, user.GetUser().GetDiscogsUserId(), entry.GetRefreshSales().GetRefreshId())
			if err != nil {
				return err
			}
			// Adjust all sale prices
			return q.b.AdjustSales(ctx, user.GetConfig().GetSaleConfig(), user, q.Enqueue)
		}

		return nil
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

		//Move records
		q.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_MoveRecords{
					MoveRecords: &pb.MoveRecords{}},
				Auth: entry.GetAuth(),
			}})

		return v
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth())
	case *pb.QueueElement_RefreshUpdates:
		return q.b.RefreshUpdates(ctx, d)
	case *pb.QueueElement_RefreshRelease:
		log.Printf("Refreshing %v for %v", entry.GetRefreshRelease().GetIid(), entry.GetRefreshRelease().GetIid())
		err := q.b.RefreshRelease(ctx, entry.GetRefreshRelease().GetIid(), d, entry.GetRefreshRelease().GetIntention() == "Manual Update")
		if err != nil {
			return err
		}
		return q.deleteRefreshMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
	case *pb.QueueElement_RefreshCollection:
		log.Printf("RefreshCollection -> %v", entry.GetRefreshCollection().GetIntention())
		return q.b.RefreshCollection(ctx, d, entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshEarliestReleaseDates:
		err := q.b.RefreshReleaseDates(ctx, d, entry.GetAuth(), entry.GetRefreshEarliestReleaseDates().GetIid(), entry.GetRefreshEarliestReleaseDates().GetMasterId(), q.Enqueue)
		if err != nil {
			return err
		}
		return q.deleteRefreshDateMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
	case *pb.QueueElement_RefreshEarliestReleaseDate:
		return q.b.RefreshReleaseDate(ctx, d, entry.GetRefreshEarliestReleaseDate().GetIid(), entry.GetRefreshEarliestReleaseDate().GetOtherRelease())
	case *pb.QueueElement_RefreshCollectionEntry:
		if q.b.ReleaseRefresh != 0 {
			return status.Errorf(codes.InvalidArgument, "There is a release running: %v", q.b.ReleaseRefresh)
		}

		if entry.GetRefreshCollectionEntry().GetPage() == 1 {
			entry.GetRefreshCollectionEntry().RefreshId = time.Now().UnixNano()
		}

		rval, err := q.b.ProcessCollectionPage(ctx, d, entry.GetRefreshCollectionEntry().GetPage(), entry.GetRefreshCollectionEntry().GetRefreshId())
		log.Printf("Processed collection page: %v %v", rval, err)

		if err != nil {
			return err
		}
		if entry.GetRefreshCollectionEntry().GetPage() == 1 {
			for i := int32(2); i <= rval; i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano() + int64(i),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{
							Page: i, RefreshId: entry.GetRefreshCollectionEntry().GetRefreshId()}},
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
			user.LastCollectionRefresh = time.Now().UnixNano()
			err = q.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to sell user: %w", err)
			}
			if err != nil {
				return err
			}

			//Move records
			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano() + int64(rval) + 10,
					Entry: &pb.QueueElement_MoveRecords{
						MoveRecords: &pb.MoveRecords{}},
					Auth: entry.GetAuth(),
				}})
			return err
		} else if entry.GetRefreshCollectionEntry().GetPage() == rval {
			return q.b.CleanCollection(ctx, q.d, entry.GetRefreshCollectionEntry().GetRefreshId())
		}

		return nil
	}

	return status.Errorf(codes.NotFound, "Unable to this handle (%t), %v -> %v", entry.GetEntry(), entry, entry.Entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	var nkeys []int64
	for _, key := range q.keys {
		if key != entry.GetRunDate() {
			nkeys = append(nkeys, key)
		}
	}
	q.keys = nkeys

	// Also delete the stored key
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, entry.GetRunDate())})
	return err
}

var (
	intention = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_intention",
		Help: "The length of the working queue I think yes",
	}, []string{"intention"})
	markerCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gramophile_marker_rejects",
		Help: "The length of the working queue I think yes",
	})
)

func (q *Queue) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	log.Printf("Enqueue: %v", req)

	// Validate entries
	switch req.GetElement().GetEntry().(type) {
	case *pb.QueueElement_RefreshRelease:
		if req.GetElement().GetRefreshRelease().GetIntention() == "" {
			intention.With(prometheus.Labels{"intention": "REJECT"}).Inc()
			return nil, status.Errorf(codes.InvalidArgument, "You must specify an intention for this refresh: %T", req.GetElement().GetEntry())
		}
		intention.With(prometheus.Labels{"intention": req.GetElement().GetRefreshRelease().GetIntention()}).Inc()

		// Check for a marker
		marker, err := q.getRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			if status.Code(err) != codes.NotFound {
				return nil, fmt.Errorf("Unable to get refresh marker: %w", err)
			}
		} else if marker > 0 && time.Since(time.Unix(0, marker)) < time.Hour*24 {
			markerCount.Inc()
			return nil, status.Errorf(codes.AlreadyExists, "Refresh is in the queue: %v", time.Since(time.Unix(0, marker)))
		}

		err = q.setRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			return nil, fmt.Errorf("Unable to write refresh marker: %w", err)
		}
	case *pb.QueueElement_RefreshEarliestReleaseDates:
		log.Printf("Trying to refresh dates: %v", req.GetElement().GetRefreshEarliestReleaseDates())
		// Check for a marker
		marker, err := q.getRefreshDateMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Printf("NO DATEMARKER")
				return nil, fmt.Errorf("Unable to get refresh datemarker: %w", err)
			}
		} else if marker > 0 && time.Since(time.Unix(0, marker)) < time.Hour*24 {
			markerCount.Inc()
			log.Printf("REJECTING because we have a refresh date in the queue")
			return nil, status.Errorf(codes.AlreadyExists, "Refresh date is in the queue: %v", time.Since(time.Unix(0, marker)))
		}

		err = q.setRefreshDateMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			return nil, fmt.Errorf("Unable to write refresh date marker: %w", err)
		}
	}

	queueAdd.With(prometheus.Labels{"type": fmt.Sprintf("%T", req.GetElement().GetEntry())}).Inc()

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
	q.keys = append(q.keys, req.GetElement().GetRunDate())

	return &pb.EnqueueResponse{}, err
}

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	t := time.Now()
	/*keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	queueLen.Set(float64(len(keys.GetKeys())))

	if len(keys.GetKeys()) == 0 {
		return nil, status.Errorf(codes.NotFound, "No queue entries")
	}

	sort.SliceStable(keys.Keys, func(i, j int) bool {
		return strings.Compare(keys.GetKeys()[i], keys.GetKeys()[j]) < 0
	})*/

	if len(q.keys) == 0 {
		return nil, status.Errorf(codes.NotFound, "No queue entries")
	}

	keys := q.keys
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	data, err := q.rstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, keys[0])})
	if err != nil {
		return nil, err
	}

	queueLen.Set(float64(len(keys)))
	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	queueLoadTime.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Observe(float64(time.Since(t).Milliseconds()))
	return entry, err
}
