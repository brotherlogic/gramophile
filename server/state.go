package server

import (
	"context"

	"github.com/brotherlogic/gramophile/config"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetState(ctx context.Context, req *pb.GetStateRequest) (*pb.GetStateResponse, error) {
	key, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	collection, err := s.d.GetRecords(ctx, key.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	return &pb.GetStateResponse{
		LastUserRefresh:    key.GetLastRefreshTime(),
		CollectionSize:     int32(len(collection)),
		LastCollectionSync: key.GetLastCollectionRefresh(),
		LastConfigUpdate:   key.GetLastConfigUpdate(),
		ConfigHash:         config.Hash(key.GetConfig()),
	}, nil
}
