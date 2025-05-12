package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) ListWantlists(ctx context.Context, _ *pb.ListWantlistsRequest) (*pb.ListWantlistsResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	lists, err := s.d.GetWantlists(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}
	return &pb.ListWantlistsResponse{Lists: lists}, nil
}

func (s *Server) GetWantlist(ctx context.Context, req *pb.GetWantlistRequest) (*pb.GetWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.d.LoadWantlist(ctx, user.GetUser().GetDiscogsUserId(), req.GetName())
	return &pb.GetWantlistResponse{List: list}, err
}
