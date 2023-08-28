package queuelogic

import (
	"context"
	"testing"

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
