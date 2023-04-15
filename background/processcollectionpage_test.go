package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"

	dpb "github.com/brotherlogic/discogs/proto"
)

func GetTestBackgroundRunner() *BackgroundRunner {
	return &BackgroundRunner{
		db: &db.TestDatabase{},
	}
}

func TestGetCollectionPage_WithNewRecord(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2})

	err := b.ProcessCollectionPage(context.Background(), d, 1)
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
