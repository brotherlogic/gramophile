package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	key, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	collection, err := s.GetRecords(ctx, key.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	return &pb.GetStateResponse{
		LastUserRefresh: key.GetLastRefreshTime(),
		CollectionSize:  len(collection),
	}, nil
}
