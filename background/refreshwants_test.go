package background

import (
	"context"
	"testing"

	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
)

func TestRefreshWant_HandlerRegistrationAndValidation(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	b.RegisterAllHandlers()

	entry := &pb.QueueElement{
		Auth: "test_auth_token",
		Entry: &pb.QueueElement_RefreshWant{
			RefreshWant: &pb.RefreshWant{
				Want: &pb.Want{
					Id: 12345,
				},
			},
		},
	}

	handler, err := b.getHandler(entry)
	if err != nil {
		t.Fatalf("handler not registered for QueueElement_RefreshWant: %v", err)
	}

	dedupKey := handler.GetDeduplicationKey(entry)
	if dedupKey != "RefreshWant-test_auth_token-12345" {
		t.Errorf("expected deduplication key 'RefreshWant-test_auth_token-12345', got '%v'", dedupKey)
	}

	err = handler.Validate(ctx, d, entry)
	if err != nil {
		t.Errorf("expected Validate to return nil, got %v", err)
	}
}
