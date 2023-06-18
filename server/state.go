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

	count := int32(0)

	maxGoroutines := 10
	guard := make(chan struct{}, maxGoroutines)
	for _, r := range collection {
		guard <- struct{}{}
		go func(r int64) {
			rec, err := s.d.GetRecord(ctx, key.GetUser().GetDiscogsUserId(), r)
			if err == nil {
				if len(rec.GetIssues()) > 0 {
					count++
				}
			}
			<-guard
		}(r)
	}

	return &pb.GetStateResponse{
		LastUserRefresh:    key.GetLastRefreshTime(),
		CollectionSize:     int32(len(collection)),
		LastCollectionSync: key.GetLastCollectionRefresh(),
		LastConfigUpdate:   key.GetLastConfigUpdate(),
		ConfigHash:         config.Hash(key.GetConfig()),
		CollectionMisses:   count,
	}, nil
}
