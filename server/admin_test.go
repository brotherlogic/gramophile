package server

import (
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestClean(t *testing.T) {
	ctx := getTestContext(123)
	d := db.NewTestDB(rstore_client.GetTestClient())
	s := Server{d: d}

	_, err := s.Clean(ctx, &pb.CleanRequest{})
	if err != nil {
		t.Errorf("Unable to run clean: %v", err)
	}
}
