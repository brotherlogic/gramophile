package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetWants(ctx context.Context, req *pb.GetWantsRequest) (*pb.GetWantsResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	wants, err := s.d.GetWants(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}
	return &pb.GetWantsResponse{Wants: wants}, nil
}

func (s *Server) AddWant(ctx context.Context, req *pb.AddWantRequest) (*pb.AddWantResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	return &pb.AddWantResponse{}, s.d.SaveWant(ctx, user.GetUser().GetDiscogsUserId(), &pb.Want{Id: req.GetWantId()})
}

func (s *Server) DeleteWant(ctx context.Context, req *pb.DeleteWantRequest) (*pb.DeleteWantResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	return &pb.DeleteWantResponse{}, s.d.DeleteWant(ctx, user, req.GetWantId())
}
