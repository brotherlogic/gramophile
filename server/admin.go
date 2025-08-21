package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) Clean(ctx context.Context, req *pb.CleanRequest) (*pb.CleanResponse, error) {
	return &pb.CleanResponse{}, s.d.Clean(ctx, req.GetType())
}
