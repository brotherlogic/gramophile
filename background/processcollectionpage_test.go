package background

import (
	"context"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"

	dpb "github.com/brotherlogic/discogs/proto"
	pbd "github.com/brotherlogic/discogs/proto"

	pstore_client "github.com/brotherlogic/pstore/client"
)

func GetTestBackgroundRunner() *BackgroundRunner {
	return &BackgroundRunner{
		db: db.NewTestDB(pstore_client.GetTestClient()),
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

func TestGetCollectionPage_WithFieldUpdates(t *testing.T) {
	b := GetTestBackgroundRunner()

	ti := time.Date(2012, time.April, 10, 0, 0, 0, 0, time.UTC)

	d := &discogs.TestDiscogsClient{UserId: 123, Fields: []*pbd.Field{{Id: 10, Name: "Cleaned"}}}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2, Notes: map[int32]string{10: ti.Format("2006-01-02")}})

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

	if record.GetLastCleanTime() != ti.Unix() {
		t.Errorf("Unable to retrieve clean time: %v (%v vs %v)", record, time.Unix(0, record.GetLastCleanTime()), ti)
	}

	if len(record.GetRelease().GetNotes()) > 0 {
		t.Errorf("Effective hanging notes here: %v", record.GetRelease())
	}
}

func TestGetCollectionPage_NotClobberingDateAdded(t *testing.T) {
	b := GetTestBackgroundRunner()
	d := &discogs.TestDiscogsClient{UserId: 123}
	d.AddCollectionRelease(&dpb.Release{InstanceId: 100, Rating: 2, Labels: []*pbd.Label{{Name: "testing"}}})
	b.db.SaveRecord(context.Background(), 123, &pb.Record{Release: &pbd.Release{Id: 1, DateAdded: 1234, Labels: []*pbd.Label{{Name: "testing"}}, InstanceId: 100}}, &db.SaveOptions{})

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

	if record.GetRelease().GetDateAdded() != 1234 {
		t.Errorf("Date has been clobbered: %v", record)
	}

	if len(record.GetRelease().GetLabels()) != 1 {
		t.Errorf("Overadded the labels: %v", record)
	}
}
