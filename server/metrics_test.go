package server

import (
	"context"
	"testing"

	"google.golang.org/grpc"
)

func TestServerTimingWithMetrics(t *testing.T) {
	s := &Server{
		trackings: make(map[string]*tracking),
	}

	// Initialize metrics
	initMetrics()

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "resp", nil
	}

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	_, err := s.ServerTiming(context.Background(), "req", info, handler)
	if err != nil {
		t.Fatalf("ServerTiming failed: %v", err)
	}
}
