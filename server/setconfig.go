package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	_, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.SetConfigResponse{}, status.Errorf(codes.FailedPrecondition, "Config could not be validated")
}
