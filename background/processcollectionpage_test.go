package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"

	dpb "github.com/brotherlogic/discogs/proto"

	rstore_client "github.com/brotherlogic/rstore/client"
)

func GetTestBackgroundRunner() *BackgroundRunner {
	return &BackgroundRunner{
		db: db.NewTestDB(rstore_client.GetTestClient()),
	}
}

func TestGetCollectionPage_WithNewRecord(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetRelease().GetRating() != 2 {
		t.Errorf("Stored record is not quite right: %v", record)
	}
}

func TestGetCollectionPage_WithDeletion(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2})

	_, err := b.ProcessCollectionPage(context.Background(), d, 1, 123)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	_, err = b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	d = &discogs.TestDiscogsClient{}
	_, err = b.ProcessCollectionPage(context.Background(), d, 1, 1234)
	if err != nil {
		t.Fatalf("Bad collection pull (2); %v", err)
	}

	err = b.CleanCollection(context.Background(), d, 1234)
	if err != nil {
		t.Fatalf("Clean failed: %v", err)
	}

	rec, err := b.db.GetRecord(context.Background(), d.GetUserId(), 100)
	if err == nil && rec.GetRelease().GetRating() == 2 {
		t.Errorf("Refresh pull did not delete record")
	}

}
