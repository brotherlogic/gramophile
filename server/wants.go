package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func convertToWant(entry *pb.WantlistEntry) *pb.Want {
	return &pb.Want{
		Id:            entry.GetId(),
		WantAddedDate: entry.GetDateEnabled(),
	}
}

func (s *Server) GetWants(ctx context.Context, req *pb.GetWantsRequest) (*pb.GetWantsResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	wants, err := s.d.GetWants(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	// Also pull wants from wantlists
	wantlists, err := s.d.GetWantlists(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	for _, list := range wantlists {
		for _, entry := range list.GetEntries() {
			found := false
			for _, already := range wants {
				if entry.GetId() == already.GetId() {
					found = true
				}
			}
			if !found {
				wants = append(wants, convertToWant(entry))
			}
		}
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
