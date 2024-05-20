package server

import (
	"context"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *Server) AddWantlist(ctx context.Context, req *pb.AddWantlistRequest) (*pb.AddWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	if user.GetConfig().GetWantsConfig().GetOrigin() == pb.WantsBasis_WANTS_DISCOGS {
		return nil, status.Errorf(codes.FailedPrecondition, "you can't add wantslist to discogs controlled wants")
	}

	return &pb.AddWantlistResponse{}, s.d.SaveWantlist(ctx, user.GetUser().GetDiscogsUserId(),
		&pb.Wantlist{
			Name:       req.GetName(),
			Type:       req.GetType(),
			Visibility: req.GetVisibility(),
		})
}

func (s *Server) GetWantlist(ctx context.Context, req *pb.GetWantlistRequest) (*pb.GetWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	list, err := s.d.LoadWantlist(ctx, user.GetUser().GetDiscogsUserId(), req.GetName())
	return &pb.GetWantlistResponse{List: list}, err
}

func (s *Server) UpdateWantlist(ctx context.Context, req *pb.UpdateWantlistRequest) (*pb.UpdateWantlistResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	list, err := s.d.LoadWantlist(ctx, user.GetUser().GetDiscogsUserId(), req.GetName())
	if err != nil {
		return nil, err
	}

	if req.GetAddId() != 0 {
		list.Entries = append(list.Entries, &pb.WantlistEntry{
			Id:    req.GetAddId(),
			Index: int32(len(list.GetEntries())) + 1,
		})

		log.Printf("SAVING %v", req.GetAddId())
		s.d.SaveWant(ctx, user.GetUser().GetDiscogsUserId(), &pb.Want{
			Id:    req.GetAddId(),
			State: pb.WantState_PENDING,
		}, "Adding from updated wantlist")
	}

	// Runs delete
	if req.GetDeleteId() != 0 {
		var entries []*pb.WantlistEntry
		for _, entry := range list.Entries {
			if entry.GetId() != req.GetDeleteId() {
				entries = append(entries, entry)
			}
		}
		list.Entries = entries
	}

	if req.GetNewType() != pb.WantlistType_TYPE_UNKNOWN {
		list.Type = req.GetNewType()
	}

	err = s.d.SaveWantlist(ctx, user.GetUser().GetDiscogsUserId(), list)
	if err != nil {
		return nil, err
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:          time.Now().UnixNano(),
			Auth:             user.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			},
		},
	})

	return &pb.UpdateWantlistResponse{}, err
}
