package server

import (
	"context"
	"fmt"
	"log"

	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	u, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("Got user: %v", u.GetUser())
	log.Printf("Down to: %v", s.di.ForUser(u.GetUser()))

	fields, err := s.di.ForUser(u.GetUser()).GetFields(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("Got fields: %v", fields)

	verr := config.ValidateConfig(ctx, fields, req.GetConfig())
	if verr != nil {
		return nil, verr
	}

	u.Config = req.GetConfig()

	// Apply the config
	keys, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, fmt.Errorf("error getting records: %w", err)
	}
	for _, key := range keys {
		r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), key)
		if err != nil {
			return nil, fmt.Errorf("error getting record from key: %v -> %w", key, err)
		}

		err = config.Apply(u.Config, r)
		if err != nil {
			return nil, err
		}

		err = s.d.SaveRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return nil, err
		}
	}

	return &pb.SetConfigResponse{}, s.d.SaveUser(ctx, u)
}
