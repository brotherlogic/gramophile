package queuelogic

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/discogs"
	ghb_client "github.com/brotherlogic/githubridge/client"
	pstore_client "github.com/brotherlogic/pstore/client"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	rspb "github.com/brotherlogic/pstore/proto"
)

type syncPstoreClient struct {
	mu     sync.Mutex
	client pstore_client.PStoreClient
}

func (s *syncPstoreClient) Read(ctx context.Context, req *rspb.ReadRequest) (*rspb.ReadResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Read(ctx, req)
}

func (s *syncPstoreClient) Write(ctx context.Context, req *rspb.WriteRequest) (*rspb.WriteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Write(ctx, req)
}

func (s *syncPstoreClient) GetKeys(ctx context.Context, req *rspb.GetKeysRequest) (*rspb.GetKeysResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.GetKeys(ctx, req)
}

func (s *syncPstoreClient) Delete(ctx context.Context, req *rspb.DeleteRequest) (*rspb.DeleteResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Delete(ctx, req)
}

func (s *syncPstoreClient) Count(ctx context.Context, req *rspb.CountRequest) (*rspb.CountResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.client.Count(ctx, req)
}

func getSyncTestClient() pstore_client.PStoreClient {
	return &syncPstoreClient{client: pstore_client.GetTestClient()}
}

func TestRunWithEmptyQueue(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueue(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

	elem, err := q.getNextEntry(context.Background())
	if err == nil {
		t.Errorf("Should have failed: %v, %v", elem, err)
	}
}

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(), "auth-token", fmt.Sprintf("%v", userid))
}

func getTestContextBeta(userid int) context.Context {
	return metadata.AppendToOutgoingContext(getTestContext(userid),
		"user-level",
		"BETA")
}

func TestMarkerCreationAndRemoval(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	di.AddCollectionRelease(&pbd.Release{InstanceId: 1234})
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{Release: &pbd.Release{InstanceId: 1234, FolderId: 12, Labels: []*pbd.Label{{Name: "AAA"}}}})
	if err != nil {
		t.Fatalf("Can't init save record: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Marker",
				},
			},
			Auth: "123",
		},
	})

	if err != nil {
		t.Fatalf("Error enqueueing: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Marker",
				},
			},
			Auth: "123",
		},
	})

	if err == nil || status.Code(err) != codes.AlreadyExists {
		t.Fatalf("Should have err'd or is not AlreadyExists: %v", err)
	}

	q.FlushQueue(ctx)

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Marker",
				},
			},
			Auth: "123",
		},
	})

	if err != nil {
		t.Errorf("Error in enqueing: %v", err)
	}
}

func TestEnqueueRefreshRelease_EmptyIntention(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	res, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid: 1234,
				},
			}}})

	if err == nil {
		t.Errorf("We were able to add with an empty intention: %v", res)
	}
}

func TestEnqueueRefreshRelease_WithIntention(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From TEst",
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Just Testing",
				},
			}}})

	if err != nil {
		t.Errorf("Unable to add refresh with intention: %v", err)
	}
}

func TestEnqueuePriority(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   200,
			Priority:  pb.QueueElement_PRIORITY_LOW,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Just Testing LOW",
				},
			}}})
	if err != nil {
		t.Fatalf("Unable to enqueue: %v", err)
	}

	_, err = q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From TEst",
			RunDate:   400,
			Priority:  pb.QueueElement_PRIORITY_HIGH,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       12345,
					Intention: "Just Testing HIGH",
				},
			}}})
	if err != nil {
		t.Fatalf("Unable to enqueue: %v", err)
	}

	entry, err := q.getNextEntry(context.Background())
	if err != nil {
		t.Fatalf("Unable to get next entry: %v", err)
	}

	if entry.GetPriority() != pb.QueueElement_PRIORITY_HIGH || entry.GetRefreshRelease().GetIntention() != "Just Testing HIGH" {
		t.Errorf("Bad element returned: %v", entry)
	}
}

func TestEnqueuePriority_Normal(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   200,
			Priority:  pb.QueueElement_PRIORITY_LOW,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "Just Testing LOW",
				},
			}}})
	if err != nil {
		t.Fatalf("Unable to enqueue: %v", err)
	}

	_, err = q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From TEst",
			RunDate:   400,
			Priority:  pb.QueueElement_PRIORITY_NORMAL,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       12345,
					Intention: "Just Testing NORMAL",
				},
			}}})
	if err != nil {
		t.Fatalf("Unable to enqueue: %v", err)
	}

	entry, err := q.getNextEntry(context.Background())
	if err != nil {
		t.Fatalf("Unable to get next entry: %v", err)
	}

	if entry.GetPriority() != pb.QueueElement_PRIORITY_NORMAL || entry.GetRefreshRelease().GetIntention() != "Just Testing NORMAL" {
		t.Errorf("Bad element returned: %v", entry)
	}
}

func TestPriorityOrdering(t *testing.T) {
	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	ctx := context.Background()

	// Enqueue LOW
	_, err := q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:  100,
			Priority: pb.QueueElement_PRIORITY_LOW,
			Entry:    &pb.QueueElement_RefreshWants{},
		}})
	if err != nil {
		t.Fatalf("Unable to enqueue LOW: %v", err)
	}

	// Enqueue NORMAL
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:  200,
			Priority: pb.QueueElement_PRIORITY_NORMAL,
			Entry:    &pb.QueueElement_RefreshWantlists{},
		}})
	if err != nil {
		t.Fatalf("Unable to enqueue NORMAL: %v", err)
	}

	// Enqueue HIGH
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:  300,
			Priority: pb.QueueElement_PRIORITY_HIGH,
			Entry:    &pb.QueueElement_RefreshCollection{},
		}})
	if err != nil {
		t.Fatalf("Unable to enqueue HIGH: %v", err)
	}

	// Should get HIGH first
	e1, err := q.getNextEntry(ctx)
	if err != nil || e1.GetPriority() != pb.QueueElement_PRIORITY_HIGH {
		t.Fatalf("Expected HIGH priority, got %v (err: %v)", e1.GetPriority(), err)
	}
	q.delete(ctx, e1)

	// Should get NORMAL second
	e2, err := q.getNextEntry(ctx)
	if err != nil || e2.GetPriority() != pb.QueueElement_PRIORITY_NORMAL {
		t.Fatalf("Expected NORMAL priority, got %v (err: %v)", e2.GetPriority(), err)
	}
	q.delete(ctx, e2)

	// Should get LOW last
	e3, err := q.getNextEntry(ctx)
	if err != nil || e3.GetPriority() != pb.QueueElement_PRIORITY_LOW {
		t.Fatalf("Expected LOW priority, got %v (err: %v)", e3.GetPriority(), err)
	}
	q.delete(ctx, e3)
}

func TestEnqueueRefreshRelease_DoubleAdd(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			}}})

	if err != nil {
		t.Errorf("Error on addition: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			}}})

	if err == nil {
		t.Errorf("Should have had error on addition: %v", err)
	}

	// But if we flush the queue it should work
	q.FlushQueue(ctx)

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			}}})

	if err != nil {
		t.Errorf("Error on addition: %v", err)
	}

}

type mockTaskHandler struct {
	err error
}

func (m *mockTaskHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return m.err
}

func (m *mockTaskHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (m *mockTaskHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func TestResourceExhausted_UserLimit(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{&pbd.Folder{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	// Register our mock task handler that returns a wrapped ResourceExhausted User Limit error
	wrappedErr := fmt.Errorf("unable to queue: %w", status.Errorf(codes.ResourceExhausted, "User queue limit reached"))
	q.b.RegisterTaskHandler("*proto.QueueElement_RefreshUser", &mockTaskHandler{err: wrappedErr})

	// Enqueue the task
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   0,
			Entry: &pb.QueueElement_RefreshUser{
				RefreshUser: &pb.RefreshUserEntry{},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Error enqueueing: %v", err)
	}

	// Run the queue in a goroutine
	go q.Run()

	// Wait and check if the task is deleted (queue length becomes 0) within 2 seconds.
	// If it slept for a minute, this will timeout and fail.
	success := false
	for i := 0; i < 20; i++ {
		list, err := q.List(ctx, &pb.ListRequest{})
		if err == nil && len(list.GetElements()) == 0 {
			success = true
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	if !success {
		t.Fatalf("Task was not deleted or queue stall triggered")
	}
}

func TestUserQueueLimit(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		Folders: []*pbd.Folder{{Name: "12 Inches", Id: 123}},
		User:    &pbd.User{DiscogsUserId: 123},
		Auth:    &pb.GramophileAuth{Token: "123"}})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	for i := 0; i < 200; i++ {
		_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Intention: "From Test",
				RunDate:   int64(i + 1),
				Entry: &pb.QueueElement_RefreshRelease{
					RefreshRelease: &pb.RefreshRelease{
						Iid:       int64(i + 1),
						Intention: fmt.Sprintf("Refresh %d", i+1),
					},
				},
				Auth: "123",
			},
		})
		if err != nil {
			t.Fatalf("Failed to enqueue item %d: %v", i+1, err)
		}
	}

	// The 201st enqueue should fail with ResourceExhausted
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "From Test",
			RunDate:   201,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       201,
					Intention: "Refresh 201",
				},
			},
			Auth: "123",
		},
	})
	if status.Code(err) != codes.ResourceExhausted {
		t.Fatalf("Expected ResourceExhausted on 201st item, got: %v", err)
	}
}

func TestDrain_JustSales_IncludesReconcileSalesAndCleansInMemory(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	// Record initial metric values
	initialQueueStateRefreshSales := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_RefreshSales"}))
	initialQueueStateReconcileSales := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_ReconcileSales"}))
	initialQueueStateRefreshRelease := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_RefreshRelease"}))
	initialQueueLenNormal := testutil.ToFloat64(queueLen.With(prometheus.Labels{"type": fmt.Sprintf("%v", pb.QueueElement_PRIORITY_NORMAL)}))

	// Enqueue 1: RefreshSales
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "Refresh Sales",
			RunDate:   1001,
			Priority:  pb.QueueElement_PRIORITY_NORMAL,
			Entry: &pb.QueueElement_RefreshSales{
				RefreshSales: &pb.RefreshSales{Page: 1},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshSales: %v", err)
	}

	// Enqueue 2: ReconcileSales
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "Reconcile Sales",
			RunDate:   1002,
			Priority:  pb.QueueElement_PRIORITY_NORMAL,
			Entry: &pb.QueueElement_ReconcileSales{
				ReconcileSales: &pb.ReconcileSales{Page: 1},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue ReconcileSales: %v", err)
	}

	// Enqueue 3: RefreshRelease (non-sales task)
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "Refresh Release",
			RunDate:   1003,
			Priority:  pb.QueueElement_PRIORITY_NORMAL,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       12345,
					Intention: "Keep this",
				},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshRelease: %v", err)
	}

	if len(q.keys) != 3 {
		t.Fatalf("Expected 3 in-memory keys before drain, got %d", len(q.keys))
	}
	if q.userCounts["123"] != 3 {
		t.Fatalf("Expected userCounts 3, got %d", q.userCounts["123"])
	}

	// Drain sales
	resp, err := q.Drain(ctx, &pb.DrainRequest{
		DrainType: pb.DrainRequest_JUST_SALES,
	})
	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}

	// Exactly 2 items should be drained (RefreshSales and ReconcileSales)
	if resp.GetCount() != 2 {
		t.Errorf("Expected Drain count to be 2, got %d", resp.GetCount())
	}

	// In-memory keys should now only have 1 item left (1003)
	if len(q.keys) != 1 || q.keys[0] != 1003 {
		t.Errorf("Expected in-memory q.keys to contain only [1003], got %v", q.keys)
	}

	// In-memory pMap should only contain 1003
	if len(q.pMap) != 1 {
		t.Errorf("Expected in-memory q.pMap to have length 1, got %v", q.pMap)
	}
	if _, ok := q.pMap[1003]; !ok {
		t.Errorf("Expected key 1003 in pMap, not found")
	}
	if _, ok := q.pMap[1001]; ok {
		t.Errorf("Expected key 1001 to be removed from pMap")
	}
	if _, ok := q.pMap[1002]; ok {
		t.Errorf("Expected key 1002 to be removed from pMap")
	}

	// User count should be decremented by 2
	if q.userCounts["123"] != 1 {
		t.Errorf("Expected userCounts['123'] == 1, got %d", q.userCounts["123"])
	}

	// Verify Prometheus metrics
	postQueueStateRefreshSales := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_RefreshSales"}))
	postQueueStateReconcileSales := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_ReconcileSales"}))
	postQueueStateRefreshRelease := testutil.ToFloat64(queueState.With(prometheus.Labels{"type": "*proto.QueueElement_RefreshRelease"}))
	postQueueLenNormal := testutil.ToFloat64(queueLen.With(prometheus.Labels{"type": fmt.Sprintf("%v", pb.QueueElement_PRIORITY_NORMAL)}))

	if postQueueStateRefreshSales != initialQueueStateRefreshSales {
		t.Errorf("Expected queueState for RefreshSales to return to %v, got %v", initialQueueStateRefreshSales, postQueueStateRefreshSales)
	}
	if postQueueStateReconcileSales != initialQueueStateReconcileSales {
		t.Errorf("Expected queueState for ReconcileSales to return to %v, got %v", initialQueueStateReconcileSales, postQueueStateReconcileSales)
	}
	if postQueueStateRefreshRelease != initialQueueStateRefreshRelease+1 {
		t.Errorf("Expected queueState for RefreshRelease to remain incremented at %v, got %v", initialQueueStateRefreshRelease+1, postQueueStateRefreshRelease)
	}
	if postQueueLenNormal != initialQueueLenNormal+1 {
		t.Errorf("Expected queueLen for NORMAL priority to be %v, got %v", initialQueueLenNormal+1, postQueueLenNormal)
	}

	// Verify pstore state: only 1003 remains
	keys, err := pstore.GetKeys(ctx, &rspb.GetKeysRequest{Prefix: QUEUE_PREFIX})
	if err != nil {
		t.Fatalf("Failed to get keys from pstore: %v", err)
	}
	if len(keys.GetKeys()) != 1 {
		t.Errorf("Expected 1 key in pstore, got %d: %v", len(keys.GetKeys()), keys.GetKeys())
	}

	// Verify getNextEntry retrieves the remaining item
	elem, err := q.getNextEntry(ctx)
	if err != nil {
		t.Fatalf("Failed to get next entry: %v", err)
	}
	if elem.GetRunDate() != 1003 || elem.GetRefreshRelease() == nil {
		t.Errorf("Unexpected next entry: %v", elem)
	}
}

func TestDrain_All(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	// Enqueue 2 items
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "Refresh Sales",
			RunDate:   2001,
			Entry: &pb.QueueElement_RefreshSales{
				RefreshSales: &pb.RefreshSales{Page: 1},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshSales: %v", err)
	}
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "Refresh Release",
			RunDate:   2002,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       12345,
					Intention: "Keep this",
				},
			},
			Auth: "123",
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshRelease: %v", err)
	}

	resp, err := q.Drain(ctx, &pb.DrainRequest{})
	if err != nil {
		t.Fatalf("Drain returned error: %v", err)
	}

	if resp.GetCount() != 2 {
		t.Errorf("Expected drain count 2, got %d", resp.GetCount())
	}
	if len(q.keys) != 0 {
		t.Errorf("Expected in-memory keys empty, got %v", q.keys)
	}
	if len(q.pMap) != 0 {
		t.Errorf("Expected in-memory pMap empty, got %v", q.pMap)
	}
	if q.userCounts["123"] != 0 {
		t.Errorf("Expected userCounts 0, got %d", q.userCounts["123"])
	}

	// getNextEntry should return NotFound
	_, err = q.getNextEntry(ctx)
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected codes.NotFound, got %v", err)
	}
}

func TestDrain_OtherSelectiveTypes(t *testing.T) {
	ctx := getTestContext(123)

	pstore := getSyncTestClient()
	d := db.NewTestDB(pstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(pstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	err := d.SaveUser(ctx, &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "123"},
	})
	if err != nil {
		t.Fatalf("Bad user: %v", err)
	}

	// Enqueue 4 items:
	// 3001: RefreshEarliestReleaseDate (RELEASE_DATES)
	// 3002: RefreshEarliestReleaseDates (RELEASE_DATES)
	// 3003: RefreshWant (WANTS)
	// 3004: RefreshCollectionEntry (REFRESH)
	tasks := []*pb.QueueElement{
		{
			Intention: "RD1",
			RunDate:   3001,
			Entry: &pb.QueueElement_RefreshEarliestReleaseDate{
				RefreshEarliestReleaseDate: &pb.RefreshEarliestReleaseDate{},
			},
			Auth: "123",
		},
		{
			Intention: "RD2",
			RunDate:   3002,
			Entry: &pb.QueueElement_RefreshEarliestReleaseDates{
				RefreshEarliestReleaseDates: &pb.RefreshEarliestReleaseDates{},
			},
			Auth: "123",
		},
		{
			Intention: "Want",
			RunDate:   3003,
			Entry: &pb.QueueElement_RefreshWant{
				RefreshWant: &pb.RefreshWant{},
			},
			Auth: "123",
		},
		{
			Intention: "RefreshCollection",
			RunDate:   3004,
			Entry: &pb.QueueElement_RefreshCollectionEntry{
				RefreshCollectionEntry: &pb.RefreshCollectionEntry{},
			},
			Auth: "123",
		},
	}

	for _, elem := range tasks {
		_, err := q.Enqueue(ctx, &pb.EnqueueRequest{Element: elem})
		if err != nil {
			t.Fatalf("Failed to enqueue %v: %v", elem, err)
		}
	}

	// 1. Drain JUST_RELEASE_DATES (should drain 3001 and 3002)
	resp, err := q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_JUST_RELEASE_DATES})
	if err != nil {
		t.Fatalf("Drain JUST_RELEASE_DATES failed: %v", err)
	}
	if resp.GetCount() != 2 {
		t.Errorf("Expected 2 drained for RELEASE_DATES, got %d", resp.GetCount())
	}
	if len(q.keys) != 2 {
		t.Errorf("Expected 2 keys remaining after RELEASE_DATES drain, got %d", len(q.keys))
	}

	// 2. Drain JUST_WANTS (should drain 3003)
	resp, err = q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_JUST_WANTS})
	if err != nil {
		t.Fatalf("Drain JUST_WANTS failed: %v", err)
	}
	if resp.GetCount() != 1 {
		t.Errorf("Expected 1 drained for WANTS, got %d", resp.GetCount())
	}
	if len(q.keys) != 1 || q.keys[0] != 3004 {
		t.Errorf("Expected only key 3004 remaining, got %v", q.keys)
	}

	// 3. Drain JUST_REFRESH (should drain 3004)
	resp, err = q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_JUST_REFRESH})
	if err != nil {
		t.Fatalf("Drain JUST_REFRESH failed: %v", err)
	}
	if resp.GetCount() != 1 {
		t.Errorf("Expected 1 drained for REFRESH, got %d", resp.GetCount())
	}
	if len(q.keys) != 0 {
		t.Errorf("Expected 0 keys remaining, got %d", len(q.keys))
	}
}


