package server

import (
	"context"
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetWants(ctx context.Context, req *pb.GetWantsRequest) (*pb.GetWantsResponse, error) {
	return nil, fmt.Errorf("Not done")
}

func (s *Server) AddWant(ctx context.Context, req *pb.AddWantRequest) (*pb.AddWantResponse, error) {
	return nil, fmt.Errorf("Not done")
}
