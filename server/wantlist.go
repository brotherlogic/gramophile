package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) AddWantlist(ctx context.Context, req *pb.AddWantlistRequest) (*pb.AddWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	return nil, s.d.SaveWantlist(ctx, user, &pb.Wantlist{Name: req.GetName()})
}

func (s *Server) GetWantlist(ctx context.Context, req *pb.GetWantlistRequest) (*pb.GetWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.d.LoadWantlist(ctx, user, req.GetName())
	return &pb.GetWantlistResponse{List: list}, err
}
