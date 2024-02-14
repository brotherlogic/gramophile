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

	_, err := q.Enqueue(context.Background(), &pb.QueueEntry{
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

	refresh, err := q.getRefreshMarker("hello", 1234)
	if err != nil {
		t.Fatalf("Error getting refresh maker")
	}
	if refresh == 0 {
		t.Fatalf("Marker was not created")
	}

	q.FlushQueue(context.Background())

	refresh, err := q.getRefreshMarker("hello", 1234)
	if err == nil || status.Code(err) != codes.NotFound {
		t.Errorf("Marker still exists (nil), or bad read: %v", err)
	}

}
