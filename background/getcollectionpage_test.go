package background

import (
	"context"
	"testing"
)

func GetTestBackgroundRunner() *BackgroundRunner {
	return &BackgroundRunner{}
}

func TestGetCollectionPage_WithNewRecord(t *testing.T) {
	b := GetTestBackgroundRunner()

	err := b.GetCollectionPage(context.Background(), 1)
	if err != nil {
		t.Errorf("Bad collection pull: %v", err)
	}

	record, err := b.db.GetRecord(context.Background(), b.user, 100)
	if err != nil {
		t.Errorf("Bad get: %v", err)
	}

	if record.GetRelease().GetRating() != 2 {
		t.Errorf("Stored record is not quite right: %v", record)
	}
}
