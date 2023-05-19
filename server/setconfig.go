package server

import (
	"context"

	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	u, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	verr := config.ValidateConfig(req.GetConfig())
	if verr != nil {
		return nil, verr
	}

	u.Config = req.GetConfig()

	// Apply the config
	keys, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}
	for _, key := range keys {
		s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), key)
	}

	return &pb.SetConfigResponse{}, s.d.SaveUser(ctx, u)
}
