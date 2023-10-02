package server

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestGenerateContext(t *testing.T) {
	ctx := generateContext(context.Background(), "blah")

	key, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		t.Fatalf("bad extract")
	}
	if key.Get("trace-id")[0] == "" {
		t.Errorf("Bad pull: %v", key)
	}
}
