package queuelogic

import (
	"context"
	"fmt"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/discogs"
	ghb_client "github.com/brotherlogic/githubridge/client"
	rstore_client "github.com/brotherlogic/rstore/client"

	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
)

func TestRunWithEmptyQueue(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

	elem, err := q.getNextEntry(context.Background())
	if err == nil {
		t.Errorf("Should have failed: %v, %v", elem, err)
	}
}

func getTestContext(userid int) context.Context {
	return metadata.AppendToOutgoingContext(context.Background(),
		"auth-token",
		fmt.Sprintf("%v", userid))
}

func TestMarkerCreationAndRemoval(t *testing.T) {
	ctx := getTestContext(123)

	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	di.AddCollectionRelease(&pbd.Release{InstanceId: 1234})
	q := GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())
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
			RunDate: 0,
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
			RunDate: 0,
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
			RunDate: 0,
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
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	res, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
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
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
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
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueueWithGHClient(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d, ghb_client.GetTestClient())

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:  200,
			Priority: pb.QueueElement_PRIORITY_LOW,
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
			RunDate:  400,
			Priority: pb.QueueElement_PRIORITY_HIGH,
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
