package queuelogic

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	ghb_client "github.com/brotherlogic/githubridge/client"
	pstore_client "github.com/brotherlogic/pstore/client"
	scraper_client "github.com/brotherlogic/scraper/client"

	pbd "github.com/brotherlogic/discogs/proto"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/pstore/proto"
)

var (
	QUEUE_PREFIX    = "gramophile/taskqueue/"
	DL_QUEUE_PREFIX = "gramophile/dlq/"

	queueLen = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_qlen",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
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
	queueElements = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_elements",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
	queueBacklogTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "gramophile_queue_backlog_time",
		Help:    "The time taken for an element to get to the front of the queue",
		Buckets: []float64{1000, 2000, 4000, 8000, 16000, 32000, 64000, 128000, 256000, 512000, 1024000, 2048000, 4096000},
	}, []string{"type", "priority"})
	queueState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_queue_state",
		Help: "The length of the working queue I think yes",
	}, []string{"type"})
)

const (
	CollectionRefresh = time.Hour * 24 * 7 // Refresh the full collection once a week
	CollectionCheck   = time.Hour * 24     // Check through collection once a day
)

type Queue struct {
	pstore    pstore_client.PStoreClient
	b         *background.BackgroundRunner
	d         discogs.Discogs
	db        db.Database
	keys      []int64
	queueMutex sync.Mutex
	pMap      map[int64]pb.QueueElement_Priority
	gclient   ghb_client.GithubridgeClient
	hMap      map[string]bool
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
		log.Printf(str, v...)
		return
	}

	prefix := fmt.Sprintf("%v: ", key)
	log.Printf(prefix+str, v...)
}

func buildContext(rt int64, t time.Duration) (context.Context, context.CancelFunc) {
	mContext := metadata.AppendToOutgoingContext(context.Background(), "queue-key", fmt.Sprintf("%v", rt))
	ctx, cancel := context.WithTimeout(mContext, t)
	return ctx, cancel
}

func GetQueue(r pstore_client.PStoreClient, b *background.BackgroundRunner, d discogs.Discogs, db db.Database) *Queue {
	gclient, err := ghb_client.GetClientInternal()
	if err != nil {
		return nil
	}
	return GetQueueWithGHClient(r, b, d, db, gclient)
}

func GetQueueWithGHClient(r pstore_client.PStoreClient, b *background.BackgroundRunner, d discogs.Discogs, db db.Database, ghc ghb_client.GithubridgeClient) *Queue {
	log.Printf("GETTING QUEUE")
	sc, err := scraper_client.GetClient()
	if err != nil {
		panic(err)
	}
	d.SetDownloader(&DownloaderBridge{scraper: sc})

	log.Printf("Loading cache")
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*5)
	defer cancel()

	keys, err := r.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		log.Fatalf("Unable to get keys: %v", err)
	}
	var ckeys []int64
	pMap := make(map[int64]pb.QueueElement_Priority)
	hMap := make(map[string]bool)
	t := time.Now()
	for _, key := range keys.GetKeys() {
		value, err := strconv.ParseInt(key[len(QUEUE_PREFIX):], 10, 64)
		if err != nil {
			log.Fatalf("Bad parse: %v (%v)", err, key)
		}
		ckeys = append(ckeys, value)
	}
	log.Printf("Loaded keys in %v", time.Since(t))

	for _, key := range ckeys {
		data, err := r.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, key)})
		if err != nil {
			if status.Code(err) != codes.NotFound {
				log.Printf("Failed to load the pmap: %v", err)
				break
			}
		}
		entry := &pb.QueueElement{}
		err = proto.Unmarshal(data.GetValue().GetValue(), entry)
		pMap[key] = entry.GetPriority()
		queueState.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Inc()

		switch t := entry.GetEntry().(type) {
		case *pb.QueueElement_RefreshWantlists:
			hMap["RefreshWantlists"] = true
		case *pb.QueueElement_RefreshCollection:
			hMap[fmt.Sprintf("RefreshCollection-%v", entry.GetAuth())] = true
		case *pb.QueueElement_RefreshCollectionEntry:
			if t.RefreshCollectionEntry.GetPage() == 1 {
				hMap[fmt.Sprintf("RefreshCollectionEntry-%v", entry.GetAuth())] = true
			}
		}
	}
	log.Printf("Loaded pmap in %v", time.Since(t))

	return &Queue{
		b: b, d: d, pstore: r, db: db, keys: ckeys, gclient: ghc,
		pMap:      pMap,
		queueMutex: sync.Mutex{},
		hMap:      hMap,
	}
}

func (q *Queue) FlushQueue(ctx context.Context) error {
	qlog(ctx, "Flushing queue")
	elem, err := q.getNextEntry(ctx)
	qlog(ctx, "First Entry: %v", elem)

	for err == nil {
		user, errv := q.db.GetUser(ctx, elem.GetAuth())
		if errv != nil {
			log.Fatalf("unable to get user to flush queue: %v -> %v from %v", errv, elem.GetAuth(), ctx)
		}
		user.User.UserSecret = user.UserSecret
		user.User.UserToken = user.UserToken
		qlog(ctx, "USER: %v", user)
		d := q.d.ForUser(user.GetUser())
		errp := q.ExecuteInternal(ctx, d, user, elem)
		qlog(ctx, "Ran %v", elem)
		if errp == nil {
			q.delete(ctx, elem)
		} else {
			qlog(ctx, "Failed to execute internal: %v -> %v", errp, elem)
			return errp
		}

		elem, err = q.getNextEntry(ctx)
		qlog(ctx, "Post flush: %v", err)
	}

	return nil
}

func (q *Queue) Run() {
	log.Printf("Running queue with %+v", q.d)
	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
		t1 := time.Now()
		entry, err := q.getNextEntry(ctx)
		if status.Code(err) == codes.NotFound {
			time.Sleep(time.Second * 2)
			cancel()
			continue
		}

		if err != nil {
			qlog(ctx, "Error getting entry: %v", err)
			time.Sleep(time.Second)
			cancel()
			continue
		}

		qlog(ctx, "Got Entry: %v (%v)", entry, time.Since(t1))

		if err == nil {
			// If the entry is in the future, wait for it
			if entry.GetRunDate() > time.Now().UnixNano() {
				sleepTime := time.Duration(entry.GetRunDate() - time.Now().UnixNano())
				if sleepTime > time.Minute {
					sleepTime = time.Minute
				}
				qlog(ctx, "Sleeping for %v (until %v)", sleepTime, time.Unix(0, entry.GetRunDate()))
				time.Sleep(sleepTime)
			}

			nctx, ncancel := buildContext(entry.GetRunDate(), time.Hour)
			user, errv := q.db.GetUser(nctx, entry.GetAuth())
			var erru error
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
				err = q.ExecuteInternal(nctx, d, user, entry)
				qlog(nctx, "Queue entry end %v in %v -> %v ", entry, time.Since(st), err)
				queueRunTime.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Observe(float64(time.Since(st).Milliseconds()))
				queueLast.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry()), "code": fmt.Sprintf("%v", status.Code(err))}).Inc()
			}
			ncancel()

			if status.Code(err) != codes.NotFound || !strings.Contains(fmt.Sprintf("%v", err), "No queue entries") {
				qlog(ctx, "Ran Entry: (%v) %v - %v [%v]", entry, err, erru, time.Since(t1))
			}

			// Back off on any type of error - unless we failed to find the user (becuase they've been deleted)
			// Or because we've run an update on something that's not found
			if err == nil || status.Code(erru) == codes.NotFound || status.Code(err) == codes.NotFound {
				q.delete(ctx, entry)
				queueState.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Dec()
			} else {
				// This is discogs throttling us
				if status.Code(err) == codes.ResourceExhausted {
					qlog(ctx, "Waiting for a minute to let our tokens regenerate")
					time.Sleep(time.Minute)
				} else if status.Code(err) == codes.Internal {
					_, err = q.gclient.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
						User:  "brotherlogic",
						Repo:  "gramophile",
						Body:  fmt.Sprintf("Internal error on gramophile queue: %v -> %v", err, entry),
						Title: "Queue Internal error",
					})
					log.Printf("Created issue -> %v", err)

					// Delete and re-enqueue with a backoff
					q.delete(ctx, entry)
					entry.RunDate = time.Now().UnixNano() + (5 * time.Minute).Nanoseconds()
					q.Enqueue(ctx, &pb.EnqueueRequest{Element: entry})
				} else {
					// Move this over to the DLQ
					data, err := proto.Marshal(entry)
					if err == nil {
						_, err = q.pstore.Write(ctx, &rspb.WriteRequest{
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

		cancel()
	}
}

func (q *Queue) Drain(ctx context.Context, req *pb.DrainRequest) (*pb.DrainResponse, error) {
	keys, err := q.pstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	for _, key := range keys.GetKeys() {
		delete := true
		if req.GetDrainType() == pb.DrainRequest_JUST_RELEASE_DATES ||
			req.GetDrainType() == pb.DrainRequest_JUST_WANTS ||
			req.GetDrainType() == pb.DrainRequest_JUST_SALES ||
			req.GetDrainType() == pb.DrainRequest_JUST_REFRESH {
			data, err := q.pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, key)})
			if err == nil {

				entry := &pb.QueueElement{}
				err = proto.Unmarshal(data.GetValue().GetValue(), entry)
				if err != nil {
					return nil, err
				}
				switch entry.Entry.(type) {
				case *pb.QueueElement_RefreshEarliestReleaseDate, *pb.QueueElement_RefreshEarliestReleaseDates:
					if req.GetDrainType() == pb.DrainRequest_JUST_RELEASE_DATES {
						delete = true
					}
				case *pb.QueueElement_RefreshWant:
					if req.GetDrainType() == pb.DrainRequest_JUST_WANTS {
						delete = true
					}
				case *pb.QueueElement_RefreshCollectionEntry:
					if req.GetDrainType() == pb.DrainRequest_JUST_REFRESH {
						delete = true
					}
				case *pb.QueueElement_RefreshSales:
					if req.GetDrainType() == pb.DrainRequest_JUST_SALES {
						delete =true
					}
				default:
					delete = false
				}
			}
		}

		if delete {
			_, err := q.pstore.Delete(ctx, &rspb.DeleteRequest{Key: key})
			log.Printf("Delete: %v", err)
		}
	}

	return &pb.DrainResponse{Count: int32(len(keys.GetKeys()))}, nil
}

func (q *Queue) List(ctx context.Context, req *pb.ListRequest) (*pb.ListResponse, error) {
	keys, err := q.pstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		return nil, err
	}

	var elems []*pb.QueueElement
	fcount := int32(0)
	for _, key := range keys.GetKeys() {
		data, err := q.pstore.Read(ctx, &rspb.ReadRequest{Key: key})
		if err != nil {
			fcount++
			continue
		}

		entry := &pb.QueueElement{}
		err = proto.Unmarshal(data.GetValue().GetValue(), entry)
		if err != nil {
			fcount++
			continue
		}
		elems = append(elems, entry)
	}

	return &pb.ListResponse{Elements: elems, SkippedCount: fcount}, nil
}

func (q *Queue) Execute(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	user, err := q.db.GetUser(ctx, req.Element.GetAuth())
	if err != nil {
		return nil, err
	}
	d := q.d.ForUser(user.GetUser())
	t := time.Now()
	err = q.ExecuteInternal(ctx, d, user, req.GetElement())
	if err == nil && time.Since(t) > time.Minute {
		resp, err := q.gclient.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
			User:  "brotherlogic",
			Repo:  "gramophile",
			Title: "Long running queue entry",
			Body:  fmt.Sprintf("%v took %v to run", req.GetElement().GetRunDate(), time.Since(t)),
		})
		if err == nil {
			q.gclient.AddLabel(ctx, &ghbpb.AddLabelRequest{
				User:  "brotherlogic",
				Repo:  "gramophile",
				Id:    int32(resp.GetIssueId()),
				Label: "investigate",
			})
		}
	}
	return &pb.EnqueueResponse{}, err
}

func generateContext(ctx context.Context, origin string) context.Context {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	hostname := "kubernetes"
	tracev := fmt.Sprintf("%v-%v-%v-%v", origin, time.Now().UnixNano(), r.Int63(), hostname)
	mContext := metadata.AppendToOutgoingContext(ctx, "trace-id", tracev)
	return mContext
}

func (q *Queue) ExecuteInternal(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement) error {
	qlog(ctx, "Queue entry start: [%v], %v", time.Since(time.Unix(0, entry.GetAdditionDate())), entry)

	if entry.GetIntention() == "" {
		q.gclient.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
			User:  "brotherlogic",
			Repo:  "gramophile",
			Body:  fmt.Sprintf("Entry %v has no intention", entry),
			Title: "Entry Missing Intention",
		})
		qlog(ctx, "DROPPING %v", entry)

		return nil
	}

	queueBacklogTime.With(prometheus.Labels{
		"type":     fmt.Sprintf("%T", entry.Entry),
		"priority": fmt.Sprintf("%v", entry.GetPriority())}).Observe(float64(time.Since(time.Unix(0, entry.GetAdditionDate())).Milliseconds()))

	queueRun.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.Entry)}).Inc()

	switch entry.Entry.(type) {
	case *pb.QueueElement_AddSale:
		nd := d.ForUser(u.GetUser())
		return q.b.AddSale(ctx, nd, entry.GetAddSale().GetInstanceId(), entry.GetAddSale().GetSaleParams(), u)
	case *pb.QueueElement_FanoutHistory:
		err := q.b.FanoutHistory(ctx, entry.GetFanoutHistory().GetType(), u, entry.GetAuth(), q.Enqueue)
		return err
	case *pb.QueueElement_RecordHistory:
		err := q.b.RecordHistory(ctx, entry.GetRecordHistory().GetType(), int64(u.GetUser().GetDiscogsUserId()), entry.GetRecordHistory().GetInstanceId())
		return err
	case *pb.QueueElement_RefreshState:
		return q.b.ProcessRefreshState(ctx, d, entry, q.Enqueue)
	case *pb.QueueElement_MoveRecord:
		return q.b.MoveRecord(ctx, d, u, entry.GetMoveRecord(), entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_MoveRecords:
		return q.b.RunMoves(ctx, u, q.Enqueue)
	case *pb.QueueElement_AddMasterWant:
		return q.b.AddMasterWant(ctx, d, entry.GetAddMasterWant().GetWant())
	case *pb.QueueElement_UpdateSale:
		return q.b.ProcessUpdateSale(ctx, d, u, entry.GetUpdateSale())
	case *pb.QueueElement_RefreshWants:
		return q.b.RefreshWants(ctx, d, entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshWant:
		return q.b.RefreshWant(ctx, d, entry.GetRefreshWant().GetWant(), entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_SyncWants:
		return q.b.ProcessSyncWants(ctx, d, u, entry, q.Enqueue)
	case *pb.QueueElement_RefreshWantlists:
		err := q.b.RefreshWantlists(ctx, d, entry.GetAuth(), q.Enqueue)
		if err == nil {
			q.queueMutex.Lock()
			q.hMap["RefreshWantlists"] = false
			q.queueMutex.Unlock()
		}
		return err
	case *pb.QueueElement_LinkSales:
		err := q.b.LinkSales(ctx, u)
		if err != nil {
			return fmt.Errorf("unable to link sales: %w", err)
		}
		q.queueMutex.Lock()
		q.hMap["LinkSales"] = false
		q.queueMutex.Unlock()
		return nil
	case *pb.QueueElement_RefreshSales:
		return q.b.ProcessRefreshSales(ctx, d, u, entry, q.Enqueue)
	case *pb.QueueElement_AddFolderUpdate:
		err := q.b.AddFolder(ctx, entry.GetAddFolderUpdate().GetFolderName(), d, u)
		if err != nil {
			return fmt.Errorf("unable to create folder: %w", err)
		}
		return nil
	case *pb.QueueElement_RefreshIntents:
		return q.b.ProcessRefreshIntents(ctx, d, entry, q.Enqueue)
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshRelease:
		return q.b.ProcessRefreshRelease(ctx, u, d, entry, q.Enqueue)
	case *pb.QueueElement_RefreshCollection:
		qlog(ctx, "RefreshCollection -> %v", entry.GetRefreshCollection().GetIntention())
		err := q.b.RefreshCollection(ctx, d, entry.GetAuth(), q.Enqueue)
		q.queueMutex.Lock()
		q.hMap[fmt.Sprintf("RefreshCollection-%v", entry.GetAuth())] = false
		q.queueMutex.Unlock()
		return err
	case *pb.QueueElement_RefreshEarliestReleaseDates:
		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return err
		}
		digWants := user.GetConfig().GetWantsConfig().GetDigitalWantList()
		err = q.b.RefreshReleaseDates(ctx, d, entry.GetAuth(), entry.GetRefreshEarliestReleaseDates().GetIid(), entry.GetRefreshEarliestReleaseDates().GetMasterId(), digWants, q.Enqueue)
		if err != nil {
			return err
		}
		return q.db.DeleteRefreshDateMarker(ctx, entry.GetAuth())
	case *pb.QueueElement_RefreshEarliestReleaseDate:
		return q.b.RefreshReleaseDate(ctx, u, d, entry.GetRefreshEarliestReleaseDate().GetUpdateDigitalWantlist(), entry.GetRefreshEarliestReleaseDate().GetIid(), entry.GetRefreshEarliestReleaseDate().GetOtherRelease(), entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshCollectionEntry:
		rintention.With(prometheus.Labels{"intention": fmt.Sprintf("%v:%v", entry.GetRefreshCollectionEntry().GetPage(), entry.GetIntention())}).Inc()
		err := q.b.ProcessRefreshCollectionEntry(ctx, d, u, entry, q.Enqueue)
		if entry.GetRefreshCollectionEntry().GetPage() == 1 {
			q.queueMutex.Lock()
			q.hMap[fmt.Sprintf("RefreshCollectionEntry-%v", entry.GetAuth())] = false
			q.queueMutex.Unlock()
		}
		return err
	case *pb.QueueElement_DeleteRecord:
		return q.b.DeleteRecord(ctx, d, entry.GetDeleteRecord().GetIid())
	}

	return status.Errorf(codes.NotFound, "Unable to this handle (%t), %v -> %v", entry.GetEntry(), entry, entry.Entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	if entry == nil {
		return nil
	}

	q.queueMutex.Lock()
	var nkeys []int64
	for _, key := range q.keys {
		if key != entry.GetRunDate() {
			nkeys = append(nkeys, key)
		}
	}
	q.keys = nkeys
	delete(q.pMap, entry.GetRunDate())
	q.queueMutex.Unlock()
	// Also delete the stored key
	_, err := q.pstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, entry.GetRunDate())})
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
	rintention = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_refresh_intention",
		Help: "The length of the working queue I think yes",
	}, []string{"intention"})
	enqueueFail = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_queue_enqueue",
	}, []string{"code"})
)

func (q *Queue) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	qlog(ctx, "Enqueue: %v", req)
	if req.GetElement().GetIntention() == "" {
		stack := debug.Stack()
		fmt.Println(string(stack))
	}

	if len(q.keys) > 100000 && req.GetElement().GetPriority() != pb.QueueElement_PRIORITY_HIGH {
		enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.ResourceExhausted)}).Inc()
		return nil, status.Errorf(codes.ResourceExhausted, "Queue is full (%v)", len(q.keys))
	}

	req.GetElement().AdditionDate = time.Now().UnixNano()

	// Validate entries
	switch req.GetElement().GetEntry().(type) {
	case *pb.QueueElement_LinkSales:
		q.queueMutex.Lock()
		if q.hMap["LinkSales"] {
			q.queueMutex.Unlock()
			// Silent fail since it's already in the queue
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
		} else {
			q.hMap["LinkSales"] = true
			q.queueMutex.Unlock()
		}
	case *pb.QueueElement_RefreshWantlists:
		q.queueMutex.Lock()
		if q.hMap["RefreshWantlists"] {
			q.queueMutex.Unlock()
			// Silent fail since it's already in the queue
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
		} else {
			q.hMap["RefreshWantlists"] = true
			q.queueMutex.Unlock()
		}
	case *pb.QueueElement_RefreshCollection:
		q.queueMutex.Lock()
		if q.hMap[fmt.Sprintf("RefreshCollection-%v", req.GetElement().GetAuth())] {
			q.queueMutex.Unlock()
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
		} else {
			q.hMap[fmt.Sprintf("RefreshCollection-%v", req.GetElement().GetAuth())] = true
			q.queueMutex.Unlock()
		}
	case *pb.QueueElement_RefreshCollectionEntry:
		if req.GetElement().GetRefreshCollectionEntry().GetPage() == 1 {
			q.queueMutex.Lock()
			if q.hMap[fmt.Sprintf("RefreshCollectionEntry-%v", req.GetElement().GetAuth())] {
				q.queueMutex.Unlock()
				enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
				return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
			} else {
				q.hMap[fmt.Sprintf("RefreshCollectionEntry-%v", req.GetElement().GetAuth())] = true
				q.queueMutex.Unlock()
			}
		}
	case *pb.QueueElement_RefreshRelease:
		if req.GetElement().GetRefreshRelease().GetIntention() == "" {
			intention.With(prometheus.Labels{"intention": "REJECT"}).Inc()
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.InvalidArgument)}).Inc()
			return nil, status.Errorf(codes.InvalidArgument, "You must specify an intention for this refresh: %T", req.GetElement().GetEntry())
		}
		intention.With(prometheus.Labels{"intention": req.GetElement().GetRefreshRelease().GetIntention()}).Inc()

		// Check for a marker
		marker, err := q.db.GetRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			if status.Code(err) != codes.NotFound {
				enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
				return nil, fmt.Errorf("Unable to get refresh marker: %w", err)
			}
		} else if marker > 0 && time.Since(time.Unix(0, marker)) < time.Hour*24 && req.GetElement().GetRefreshRelease().GetIntention() != "Manual Update" {
			markerCount.Inc()
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return nil, status.Errorf(codes.AlreadyExists, "Refresh is in the queue: %v", time.Since(time.Unix(0, marker)))
		}

		err = q.db.SetRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
		if err != nil {
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
			return nil, fmt.Errorf("Unable to write refresh marker: %w", err)
		}
	}

	queueAdd.With(prometheus.Labels{"type": fmt.Sprintf("%T", req.GetElement().GetEntry())}).Inc()

	data, err := proto.Marshal(req.GetElement())
	if err != nil {
		return nil, err
	}
	_, err = q.pstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("%v%v", QUEUE_PREFIX, req.GetElement().GetRunDate()),
		Value: &anypb.Any{Value: data},
	})

	if err == nil {
		queueLen.With(prometheus.Labels{"type": fmt.Sprintf("%v", req.GetElement().GetPriority())}).Inc()
		queueState.With(prometheus.Labels{"type": fmt.Sprintf("%T", req.GetElement().GetEntry())}).Inc()
		q.queueMutex.Lock()
		q.keys = append(q.keys, req.GetElement().GetRunDate())
		q.pMap[req.GetElement().GetRunDate()] = req.GetElement().GetPriority()
		q.queueMutex.Unlock()
	}
	qlog(ctx, "Adding %v", req)
	qlog(ctx, "Appended %v -> %v [%v]", req.GetElement(), len(q.keys), req.GetElement().GetRunDate())

	enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
	return &pb.EnqueueResponse{}, err
}

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	t := time.Now()
	counts := make(map[string]float64)
	q.queueMutex.Lock()
	for _, val := range q.pMap {
		counts[fmt.Sprintf("%v", val)]++
	}
	for str, val := range counts {
		queueLen.With(prometheus.Labels{"type": str}).Set(float64(val))
	}
	if len(q.keys) == 0 {
		q.queueMutex.Unlock()
		return nil, status.Errorf(codes.NotFound, "No queue entries")
	}

	keys := append([]int64(nil), q.keys...)
	q.queueMutex.Unlock()

	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	q.queueMutex.Lock()
	foundKey := int64(-1)
	for _, key := range keys {
		if val, ok := q.pMap[key]; ok && val == pb.QueueElement_PRIORITY_HIGH {
			foundKey = key
			break
		}
	}

	if foundKey == -1 {
		for _, key := range keys {
			if val, ok := q.pMap[key]; ok && val == pb.QueueElement_PRIORITY_NORMAL {
				foundKey = key
				break
			}
		}
	}

	if foundKey == -1 {
		foundKey = keys[0]
	}
	q.queueMutex.Unlock()

	data, err := q.pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, foundKey)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			q.queueMutex.Lock()
			var nkeys []int64
			for _, key := range q.keys {
				if key != foundKey {
					nkeys = append(nkeys, key)
				}
			}
			q.keys = nkeys
			delete(q.pMap, foundKey)
			q.queueMutex.Unlock()
		}
		return nil, err
	}

	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	queueLoadTime.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Observe(float64(time.Since(t).Milliseconds()))
	return entry, err
}
