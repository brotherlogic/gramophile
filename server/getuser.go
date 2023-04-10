package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) GetUser(ctx context.Context, _ *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Unable to authenticate")
	}
	return &pb.GetUserResponse{User: user}, nil
}

func (s *Server) GetUsers(ctx context.Context, _ *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	keys, err := s.d.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.GetUsersResponse{UserIds: keys}, nil
}
