package queuelogic

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
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
	pMapMutex sync.Mutex
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

		switch entry.GetEntry().(type) {
		case *pb.QueueElement_RefreshWantlists:
			hMap["RefreshWantlists"] = true
		}
	}
	log.Printf("Loaded pmap in %v", time.Since(t))

	return &Queue{
		b: b, d: d, pstore: r, db: db, keys: ckeys, gclient: ghc,
		pMap:      pMap,
		pMapMutex: sync.Mutex{},
		hMap:      hMap,
	}
}

func (q *Queue) getRefreshMarker(ctx context.Context, user string, id int64) (int64, error) {
	entry, err := q.pstore.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id)})

	if err != nil {
		return -1, err
	}

	return int64(binary.BigEndian.Uint64(entry.GetValue().GetValue())), nil
}

func (q *Queue) getRefreshDateMarker(ctx context.Context, user string) (int64, error) {
	entry, err := q.pstore.Read(ctx, &rspb.ReadRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v", user)})

	if err != nil {
		return -1, err
	}

	return int64(binary.BigEndian.Uint64(entry.GetValue().GetValue())), nil
}

func (q *Queue) setRefreshMarker(ctx context.Context, user string, id int64) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	_, err := q.pstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id),
		Value: &anypb.Any{Value: b},
	})
	qlog(ctx, "Setting %v -> %v", fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id), err)

	if err != nil {
		return err
	}

	return nil
}

func (q *Queue) setRefreshDateMarker(ctx context.Context, user string) error {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(time.Now().UnixNano()))
	_, err := q.pstore.Write(ctx, &rspb.WriteRequest{
		Key:   fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v", user),
		Value: &anypb.Any{Value: b},
	})

	if err != nil {
		return err
	}

	return nil
}

func (q *Queue) deleteRefreshMarker(ctx context.Context, user string, id int64) error {
	_, err := q.pstore.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id),
	})
	qlog(ctx, "Deleting %v -> %v", fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release/%v-%v", user, id), err)

	return err
}

func (q *Queue) deleteRefreshDateMarker(ctx context.Context, user string) error {
	_, err := q.pstore.Delete(ctx, &rspb.DeleteRequest{
		Key: fmt.Sprintf("github.com/brotherlogic/gramophile/refresh_release_date/%v", user),
	})

	return err
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
		if status.Code(err) != codes.NotFound {
			qlog(ctx, "Got Entry: %v and %v (%v)", entry, err, time.Since(t1))
		}
		var erru error
		if err == nil {
			nctx, ncancel := buildContext(entry.GetRunDate(), time.Hour)
			user, errv := q.db.GetUser(nctx, entry.GetAuth())
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
		}

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

				entry.RunDate += (5 * time.Minute).Nanoseconds()
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

func (q *Queue) ExecuteInternal(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement) error {
	qlog(ctx, "Queue entry start: [%v], %v", time.Since(time.Unix(0, entry.GetAdditionDate())), entry)

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
		err := q.b.RefreshState(ctx, entry.GetRefreshState().GetIid(), d, entry.GetRefreshState().GetForce())
		if err != nil {
			if status.Code(err) == codes.NotFound {
				q.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						Auth:      entry.GetAuth(),
						Force:     true,
						RunDate:   time.Now().UnixNano(),
						Intention: fmt.Sprintf("Refreshing collection from release state %v", entry.GetRefreshState().GetIid()),
						Entry: &pb.QueueElement_RefreshCollectionEntry{
							RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
						},
					},
				})
			}
		}
		return err
	case *pb.QueueElement_MoveRecord:
		rec, err := q.db.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), entry.GetMoveRecord().GetRecordIid())
		if err != nil {
			return fmt.Errorf("unable to get record: %w", err)
		}

		fNum := int32(-1)
		for _, folder := range u.GetFolders() {
			if folder.GetName() == entry.GetMoveRecord().GetMoveFolder() {
				fNum = folder.GetId()
			}
		}

		log.Printf("Moving record from %v to %v", rec.GetRelease().GetFolderId(), fNum)

		// Fast exit if we don't need to make this move
		if rec.GetRelease().GetFolderId() == fNum {
			return nil
		}

		if fNum < 0 {
			return status.Errorf(codes.NotFound, "folder %v was not found", entry.GetMoveRecord().GetMoveFolder())
		}

		err = d.SetFolder(ctx, rec.GetRelease().GetInstanceId(), rec.GetRelease().GetId(), rec.GetRelease().GetFolderId(), fNum)
		if err != nil {
			return fmt.Errorf("unable to move record: %w", err)
		}

		qlog(ctx, "Setting folder: %v", fNum)

		//Update and save record
		rec.GetRelease().FolderId = int32(fNum)
		err = q.db.SaveRecordWithUpdate(ctx, u.GetUser().GetDiscogsUserId(), rec, &pb.RecordUpdate{
			Date: time.Now().UnixNano(),
			//Explanation: []string{fmt.Sprintf("Moved to %v following rule %v", entry.GetMoveRecord().GetMoveFolder(), entry.GetMoveRecord().GetRule())},
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
			qlog(ctx, "Skipping %v", entry)
			return nil
		}
		err := q.b.UpdateSalePrice(ctx, d, entry.GetUpdateSale().GetSaleId(), entry.GetUpdateSale().GetReleaseId(), entry.GetUpdateSale().GetCondition(), entry.GetUpdateSale().GetNewPrice(), entry.GetUpdateSale().GetMotivation())
		qlog(ctx, "Updated sale price for %v -> %v", entry.GetUpdateSale().GetSaleId(), err)

		// Not Found means the sale was deleted - if so remove from the db
		if status.Code(err) == codes.NotFound {
			qlog(ctx, "Deleting sale for %v (%v) since we can't locate the sale", entry.GetUpdateSale().GetReleaseId(), entry.GetUpdateSale().GetSaleId())
			return q.db.DeleteSale(ctx, u.GetUser().GetDiscogsUserId(), entry.GetUpdateSale().GetSaleId())
		}
		return err
	case *pb.QueueElement_RefreshWants:
		return q.b.RefreshWants(ctx, d, entry.GetAuth(), q.Enqueue)
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
			err = q.b.AlignWants(ctx, d, user.GetConfig().GetWantsConfig())
			if err != nil {
				return err
			}

			// Save any dirty wants
			wants, err := q.db.GetWants(ctx, user.GetUser().GetDiscogsUserId())
			if err != nil {
				return err
			}
			for _, want := range wants {
				if !want.GetClean() {
					_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
						Element: &pb.QueueElement{
							RunDate:          time.Now().UnixNano(),
							Auth:             user.GetAuth().GetToken(),
							BackoffInSeconds: 60,
							Entry: &pb.QueueElement_RefreshWant{
								RefreshWant: &pb.RefreshWant{
									Want: want,
								},
							},
						},
					})
				}
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
		err := q.b.RefreshWantlists(ctx, d, entry.GetAuth(), q.Enqueue)
		if err == nil {
			q.hMap["RefreshWantlists"] = false
		}
		return err
	case *pb.QueueElement_LinkSales:
		err := q.b.LinkSales(ctx, u)
		if err != nil {
			return fmt.Errorf("unable to link sales: %w", err)
		}
		q.hMap["LinkSales"] = false
		return nil
	case *pb.QueueElement_RefreshSales:
		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return fmt.Errorf("unable to get user: %w", err)
		}

		if time.Since(time.Unix(0, user.GetLastSaleRefresh())) < time.Hour*24 && !entry.GetForce() {
			qlog(ctx, "Skipping refreshRefreshSales sales because %v", time.Since(time.Unix(0, user.GetLastSaleRefresh())))
			return nil
		}

		if entry.GetRefreshSales().GetPage() == 1 {
			entry.GetRefreshSales().RefreshId = time.Now().UnixNano()
		}
		pages, err := q.b.SyncSales(ctx, d, entry.GetRefreshSales().GetPage(), entry.GetRefreshSales().GetRefreshId())

		if err != nil {
			return err
		}

		qlog(ctx, "Got user: %v with %v", user, entry.GetRefreshSales())

		if entry.GetRefreshSales().GetPage() == 1 {
			for i := int32(2); i <= pages.GetPages(); i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano() + int64(i),
					Force:   entry.GetForce(),
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
		}

		qlog(ctx, "Checking for Clean %v vs %v", entry.GetRefreshSales().GetPage(), pages.GetPages())
		if entry.GetRefreshSales().GetPage() >= pages.GetPages() {
			user.LastSaleRefresh = time.Now().UnixNano()
			err = q.db.SaveUser(ctx, user)
			if err != nil {
				return fmt.Errorf("unable to sell user: %w", err)
			}
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
		i, err := q.db.GetIntent(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId(), entry.GetRefreshIntents().GetTimestamp())
		if err != nil {
			return fmt.Errorf("unable to get intent: %w", err)
		}
		v := q.b.ProcessIntents(ctx, d, r, i, entry.GetAuth(), q.Enqueue)
		if v != nil {
			return v
		}
		qlog(ctx, "Processed intent (%v) -> %v", i, v)

		//Move records
		q.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate: time.Now().UnixNano(),
				Entry: &pb.QueueElement_MoveRecords{
					MoveRecords: &pb.MoveRecords{}},
				Auth: entry.GetAuth(),
			}})

		return q.db.DeleteIntent(ctx, d.GetUserId(), entry.GetRefreshIntents().GetInstanceId(), entry.GetRefreshIntents().GetTimestamp())
	case *pb.QueueElement_RefreshUser:
		return q.b.RefreshUser(ctx, d, entry.GetRefreshUser().GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshRelease:
		err := q.b.RefreshRelease(ctx, entry.GetRefreshRelease().GetIid(), d, entry.GetForce() || entry.GetRefreshRelease().GetIntention() == "Manual Update")
		qlog(ctx, "Refreshing %v for %v -> %v", entry.GetRefreshRelease().GetIid(), entry.GetRefreshRelease().GetIid(), err)
		if err != nil {
			if status.Code(err) == codes.NotFound {
				q.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:   time.Now().UnixNano(),
						Intention: fmt.Sprintf("Refreshing collection from release release %v", entry.GetRefreshRelease().GetIid()),
						Entry: &pb.QueueElement_RefreshCollectionEntry{
							RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
						},
					},
				})
			}
		}
		derr := q.deleteRefreshMarker(ctx, entry.GetAuth(), entry.GetRefreshRelease().GetIid())
		if derr != nil {
			return err
		}
		return derr
	case *pb.QueueElement_RefreshCollection:
		qlog(ctx, "RefreshCollection -> %v", entry.GetRefreshCollection().GetIntention())
		return q.b.RefreshCollection(ctx, d, entry.GetAuth(), q.Enqueue)
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
		return q.deleteRefreshDateMarker(ctx, entry.GetAuth())
	case *pb.QueueElement_RefreshEarliestReleaseDate:
		return q.b.RefreshReleaseDate(ctx, d, entry.GetRefreshEarliestReleaseDate().GetUpdateDigitalWantlist(), entry.GetRefreshEarliestReleaseDate().GetIid(), entry.GetRefreshEarliestReleaseDate().GetOtherRelease(), entry.GetAuth(), q.Enqueue)
	case *pb.QueueElement_RefreshCollectionEntry:
		rintention.With(prometheus.Labels{"intention": fmt.Sprintf("%v:%v", entry.GetRefreshCollectionEntry().GetPage(), entry.GetIntention())}).Inc()
		user, err := q.db.GetUser(ctx, entry.GetAuth())
		if err != nil {
			return fmt.Errorf("unable to get user: %w", err)
		}

		if entry.GetRefreshCollectionEntry().GetPage() == 1 {
			entry.GetRefreshCollectionEntry().RefreshId = time.Now().UnixNano()
		}

		rval, err := q.b.ProcessCollectionPage(ctx, d, entry.GetRefreshCollectionEntry().GetPage(), entry.GetRefreshCollectionEntry().GetRefreshId())
		qlog(ctx, "Processed collection page: %v %v", rval, err)

		if err != nil {
			return err
		}
		if entry.GetRefreshCollectionEntry().GetPage() == 1 {

			for i := int32(2); i <= rval; i++ {
				_, err = q.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
					Force:     entry.GetForce(),
					RunDate:   time.Now().UnixNano() + int64(i),
					Intention: entry.GetIntention(),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{
							Page: i, RefreshId: entry.GetRefreshCollectionEntry().GetRefreshId()}},
					Auth: entry.GetAuth(),
				}})
				if err != nil {
					return fmt.Errorf("unable to enqueue: %w", err)
				}
				user.LastCollectionRefresh = time.Now().UnixNano()
				err = q.db.SaveUser(ctx, user)
				if err != nil {
					return fmt.Errorf("unable to sell user: %w", err)
				}
			}
		} else if entry.GetRefreshCollectionEntry().GetPage() == rval {
			qlog(ctx, "Writing collection refresh chip")
			//Move records
			_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Force:   entry.GetForce(),
					RunDate: time.Now().UnixNano() + int64(rval) + 10,
					Entry: &pb.QueueElement_MoveRecords{
						MoveRecords: &pb.MoveRecords{}},
					Auth: entry.GetAuth(),
				}})
			qlog(ctx, "Found %v", err)
			if err != nil {
				return err
			}
			return q.b.CleanCollection(ctx, q.d.ForUser(user.GetUser()), entry.GetRefreshCollectionEntry().GetRefreshId())
		}
		return nil
	}

	return status.Errorf(codes.NotFound, "Unable to this handle (%t), %v -> %v", entry.GetEntry(), entry, entry.Entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	if entry == nil {
		return nil
	}

	var nkeys []int64
	for _, key := range q.keys {
		if key != entry.GetRunDate() {
			nkeys = append(nkeys, key)
		}
	}
	q.keys = nkeys
	q.pMapMutex.Lock()
	delete(q.pMap, entry.GetRunDate())
	q.pMapMutex.Unlock()
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

	if len(q.keys) > 100000 && req.GetElement().GetPriority() != pb.QueueElement_PRIORITY_HIGH {
		enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.ResourceExhausted)}).Inc()
		return nil, status.Errorf(codes.ResourceExhausted, "Queue is full (%v)", len(q.keys))
	}

	req.GetElement().AdditionDate = time.Now().UnixNano()

	// Validate entries
	switch req.GetElement().GetEntry().(type) {
	case *pb.QueueElement_LinkSales:
		if q.hMap["LinkSales"] {
			// Silent fail since it's already in the queue
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
		} else {
			q.hMap["LinkSales"] = true
		}
	case *pb.QueueElement_RefreshWantlists:
		if q.hMap["RefreshWantlists"] {
			// Silent fail since it's already in the queue
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.AlreadyExists)}).Inc()
			return &pb.EnqueueResponse{}, status.Errorf(codes.AlreadyExists, "Already have %v in the queue", req.GetElement().GetEntry())
		} else {
			q.hMap["RefreshWantlists"] = true
		}
	case *pb.QueueElement_RefreshRelease:
		if req.GetElement().GetRefreshRelease().GetIntention() == "" {
			intention.With(prometheus.Labels{"intention": "REJECT"}).Inc()
			enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", codes.InvalidArgument)}).Inc()
			return nil, status.Errorf(codes.InvalidArgument, "You must specify an intention for this refresh: %T", req.GetElement().GetEntry())
		}
		intention.With(prometheus.Labels{"intention": req.GetElement().GetRefreshRelease().GetIntention()}).Inc()

		// Check for a marker
		marker, err := q.getRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
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

		err = q.setRefreshMarker(ctx, req.Element.GetAuth(), req.GetElement().GetRefreshRelease().GetIid())
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
	}
	q.keys = append(q.keys, req.GetElement().GetRunDate())
	qlog(ctx, "Adding %v", req)
	q.pMapMutex.Lock()
	q.pMap[req.GetElement().GetRunDate()] = req.GetElement().GetPriority()
	q.pMapMutex.Unlock()
	qlog(ctx, "Appended %v -> %v [%v]", req.GetElement(), len(q.keys), req.GetElement().GetRunDate())

	enqueueFail.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
	return &pb.EnqueueResponse{}, err
}

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	t := time.Now()
	/*keys, err := q.pstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
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
	counts := make(map[string]float64)
	q.pMapMutex.Lock()
	for _, val := range q.pMap {
		counts[fmt.Sprintf("%v", val)]++
	}
	q.pMapMutex.Unlock()
	for str, val := range counts {
		queueLen.With(prometheus.Labels{"type": str}).Set(float64(val))
	}
	if len(q.keys) == 0 {
		return nil, status.Errorf(codes.NotFound, "No queue entries")
	}

	keys := q.keys
	sort.SliceStable(keys, func(i, j int) bool {
		return keys[i] < keys[j]
	})

	foundKey := keys[0]

	q.pMapMutex.Lock()
	if val, ok := q.pMap[foundKey]; ok && val != pb.QueueElement_PRIORITY_HIGH {
		// Find a better one
		for _, key := range keys {
			if val, ok := q.pMap[key]; ok && val == pb.QueueElement_PRIORITY_HIGH {
				log.Printf("Found a P_H entry: %v", key)
				foundKey = key
				break
			}
		}

		log.Printf("Unable to locate P_H entry from %v entries", len(q.pMap))
	} else {
		log.Printf("pMasp error: %v", len(q.pMap))
	}
	q.pMapMutex.Unlock()

	data, err := q.pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%v", QUEUE_PREFIX, foundKey)})
	if err != nil {
		if status.Code(err) == codes.NotFound {
			q.keys = keys[1:]
		}
		return nil, err
	}

	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	queueLoadTime.With(prometheus.Labels{"type": fmt.Sprintf("%T", entry.GetEntry())}).Observe(float64(time.Since(t).Milliseconds()))
	return entry, err
}
