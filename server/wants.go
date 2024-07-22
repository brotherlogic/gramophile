package server

import (
	"context"
	"log"
	"time"

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

	if req.GetReleaseId() > 0 {
		want, err := s.d.GetWant(ctx, user.GetUser().GetDiscogsUserId(), req.GetReleaseId())
		if err != nil {
			return nil, err
		}

		var updates []*pb.Update
		if req.GetIncludeUpdates() {
			updates, err = s.d.GetWantUpdates(ctx, user.User.GetDiscogsUserId(), req.GetReleaseId())
			if err != nil {
				return nil, err
			}
		}

		return &pb.GetWantsResponse{
			Wants: []*pb.WantResponse{
				{
					Want:    want,
					Updates: updates,
				},
			},
		}, nil
	}

	wants, err := s.d.GetWants(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	var responses []*pb.WantResponse

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

	for _, want := range wants {
		var updates []*pb.Update
		if req.GetIncludeUpdates() {
			updates, err = s.d.GetWantUpdates(ctx, user.GetUser().GetDiscogsUserId(), want.GetId())
			if err != nil {
				return nil, err
			}
		}
		responses = append(responses, &pb.WantResponse{
			Want:    want,
			Updates: updates,
		})
	}

	return &pb.GetWantsResponse{Wants: responses}, nil
}

func (s *Server) AddWant(ctx context.Context, req *pb.AddWantRequest) (*pb.AddWantResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	nw := &pb.Want{
		State:        pb.WantState_WANTED,
		Id:           req.GetWantId(),
		MasterId:     req.GetMasterWantId(),
		MasterFilter: req.GetFilter()}

	err = s.d.SaveWant(
		ctx,
		user.GetUser().GetDiscogsUserId(),
		nw,
		"Added from API")
	log.Printf("SAVED %v (%v)", err, user.GetUser().GetDiscogsUserId())

	// Enqueue an updae
	s.qc.Enqueue(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
		RunDate:          time.Now().UnixNano(),
		Auth:             user.GetAuth().GetToken(),
		BackoffInSeconds: 60,
		Entry: &pb.QueueElement_RefreshWant{
			RefreshWant: &pb.RefreshWant{
				Want: nw,
			},
		},
	}})

	return &pb.AddWantResponse{}, err
}
