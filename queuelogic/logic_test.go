package queuelogic

import (
	"context"
	"testing"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/discogs"
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

func TestMarkerCreationAndRemoval(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{}
	q := GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

	_, err := q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "What",
				},
			},
			Auth: "hello",
		},
	})

	if err != nil {
		t.Fatalf("Error enqueueing: %v", err)
	}

	_, err = q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "What",
				},
			},
			Auth: "hello",
		},
	})

	if err == nil || status.Code(err) != codes.AlreadyExists {
		t.Fatalf("Should have err'd or is not AlreadyExists: %v", err)
	}

	q.FlushQueue(context.Background())

	_, err = q.Enqueue(context.Background(), &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: 0,
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       1234,
					Intention: "What",
				},
			},
			Auth: "hello",
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
	q := GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

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
	q := GetQueue(rstore, background.GetBackgroundRunner(d, "", "", ""), di, d)

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
