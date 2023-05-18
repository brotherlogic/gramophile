package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/protobuf/proto"
)

func (s *Server) SetIntent(ctx context.Context, req *pb.SetIntentRequest) (*pb.SetIntentResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	exint, err := s.d.GetIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		return nil, err
	}

	// Merge in the proto def
	proto.Merge(exint, req.GetIntent())

	return &pb.SetIntentResponse{}, s.d.SaveIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId(), exint)
}
