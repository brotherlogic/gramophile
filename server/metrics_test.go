package server

import (
	"context"
	"sync"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
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

func TestServerTimingConcurrent(t *testing.T) {
	s := &Server{
		trackings: make(map[string]*tracking),
	}

	initMetrics()

	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/Method",
	}

	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		s.Observe(ctx, "step")
		return "resp", nil
	}

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("tracking-uuid", "test-uuid"))
			_, err := s.ServerTiming(ctx, "req", info, handler)
			if err != nil {
				t.Errorf("ServerTiming failed: %v", err)
			}
		}()
	}
	wg.Wait()
}

