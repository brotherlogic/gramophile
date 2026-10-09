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

func TestDrainJustSales(t *testing.T) {
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

	// Enqueue RefreshSales
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "sales refresh",
			RunDate:   1001,
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshSales{
				RefreshSales: &pb.RefreshSales{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshSales: %v", err)
	}

	// Enqueue ReconcileSales
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "reconcile sales",
			RunDate:   1002,
			Auth:      "123",
			Entry: &pb.QueueElement_ReconcileSales{
				ReconcileSales: &pb.ReconcileSales{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue ReconcileSales: %v", err)
	}

	// Enqueue MoveRecords
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "move records",
			RunDate:   1003,
			Auth:      "123",
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue MoveRecords: %v", err)
	}

	// Enqueue RefreshWant
	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "refresh want",
			RunDate:   1004,
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshWant{
				RefreshWant: &pb.RefreshWant{Want: &pb.Want{Id: 999}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshWant: %v", err)
	}

	if len(q.keys) != 4 {
		t.Fatalf("Expected 4 keys in queue, got %d", len(q.keys))
	}

	// Drain JUST_SALES
	resp, err := q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_JUST_SALES})
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if resp.GetCount() != 2 {
		t.Errorf("Expected 2 drained items, got %d", resp.GetCount())
	}

	// In-memory keys should now only have MoveRecords and RefreshWant
	if len(q.keys) != 2 {
		t.Errorf("Expected 2 remaining keys in memory, got %d (%v)", len(q.keys), q.keys)
	}
	if _, ok := q.pMap[1001]; ok {
		t.Errorf("Expected runDate 1001 removed from pMap")
	}
	if _, ok := q.pMap[1002]; ok {
		t.Errorf("Expected runDate 1002 removed from pMap")
	}
	if _, ok := q.pMap[1003]; !ok {
		t.Errorf("Expected runDate 1003 preserved in pMap")
	}
	if _, ok := q.pMap[1004]; !ok {
		t.Errorf("Expected runDate 1004 preserved in pMap")
	}

	// MoveRecords should still be readable in pstore
	data, err := pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%d", QUEUE_PREFIX, 1003)})
	if err != nil || data == nil {
		t.Errorf("Expected MoveRecords still in pstore, got err: %v", err)
	}

	// RefreshSales and ReconcileSales should be deleted from pstore
	_, err = pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%d", QUEUE_PREFIX, 1001)})
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected RefreshSales deleted from pstore, got err: %v", err)
	}
	_, err = pstore.Read(ctx, &rspb.ReadRequest{Key: fmt.Sprintf("%v%d", QUEUE_PREFIX, 1002)})
	if status.Code(err) != codes.NotFound {
		t.Errorf("Expected ReconcileSales deleted from pstore, got err: %v", err)
	}
}

func TestDrainJustWants(t *testing.T) {
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

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "refresh want",
			RunDate:   2001,
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshWant{
				RefreshWant: &pb.RefreshWant{Want: &pb.Want{Id: 1}},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshWant: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "move records",
			RunDate:   2002,
			Auth:      "123",
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue MoveRecords: %v", err)
	}

	resp, err := q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_JUST_WANTS})
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if resp.GetCount() != 1 {
		t.Errorf("Expected 1 drained item, got %d", resp.GetCount())
	}
	if len(q.keys) != 1 || q.keys[0] != 2002 {
		t.Errorf("Expected only key 2002 remaining, got %v", q.keys)
	}
}

func TestDrainAll(t *testing.T) {
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

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "move records",
			RunDate:   3001,
			Auth:      "123",
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue MoveRecords: %v", err)
	}

	_, err = q.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Intention: "sales refresh",
			RunDate:   3002,
			Auth:      "123",
			Entry: &pb.QueueElement_RefreshSales{
				RefreshSales: &pb.RefreshSales{Page: 1},
			},
		},
	})
	if err != nil {
		t.Fatalf("Failed to enqueue RefreshSales: %v", err)
	}

	resp, err := q.Drain(ctx, &pb.DrainRequest{DrainType: pb.DrainRequest_UNKNOWN})
	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if resp.GetCount() != 2 {
		t.Errorf("Expected 2 drained items, got %d", resp.GetCount())
	}
	if len(q.keys) != 0 {
		t.Errorf("Expected 0 keys remaining, got %d", len(q.keys))
	}
	if len(q.pMap) != 0 {
		t.Errorf("Expected empty pMap, got %v", q.pMap)
	}
}

