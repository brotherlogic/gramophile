package server

import (
	"context"
	"log"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) UpgradeUser(ctx context.Context, req *pb.UpgradeUserRequest) (*pb.UpgradeUserResponse, error) {
	resp, err := s.GetUsers(ctx, &pb.GetUsersRequest{})
	if err != nil {
		return nil, err
	}

	for _, user := range resp.GetUsers() {
		if user.GetUser().GetUsername() == req.GetUsername() {
			user.State = req.GetNewState()
			return &pb.UpgradeUserResponse{}, s.d.SaveUser(ctx, user)
		}
	}

	return &pb.UpgradeUserResponse{}, status.Errorf(codes.NotFound, "User %v was not found", req.GetUsername())
}

func (s *Server) DeleteUser(ctx context.Context, req *pb.DeleteUserRequest) (*pb.DeleteUserResponse, error) {
	if req.GetSoftDelete() {
		return &pb.DeleteUserResponse{}, s.d.DeleteUserData(ctx, req.GetId())
	}
	return &pb.DeleteUserResponse{}, s.d.DeleteUser(ctx, req.GetId())
}

func (s *Server) GetUser(ctx context.Context, _ *pb.GetUserRequest) (*pb.GetUserResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated, "Unable to authenticate")
	}
	return &pb.GetUserResponse{User: user}, nil
}

func (s *Server) GetUsers(ctx context.Context, req *pb.GetUsersRequest) (*pb.GetUsersResponse, error) {
	keys, err := s.d.GetUsers(ctx)
	if err != nil {
		return nil, err
	}

	var users []*pb.StoredUser
	for _, key := range keys {
		log.Printf("KEY: %v", key)
		user, err := s.d.GetUser(ctx, key)
		if err != nil {
			return nil, err
		}
		if req.GetState() == pb.StoredUser_USER_STATE_UNKNOWN || user.GetState() != req.GetState() {
			users = append(users, user)
		}
	}

	return &pb.GetUsersResponse{Users: users}, nil
}
