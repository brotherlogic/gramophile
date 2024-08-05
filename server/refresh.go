package server

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) RefreshRecord(ctx context.Context, req *pb.RefreshRecordRequest) (*pb.RefreshRecordResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting user: %w", err)
	}

	if !req.JustState {
		_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Force:   true,
				RunDate: time.Now().UnixNano(),
				Auth:    user.GetAuth().GetToken(),
				Entry: &pb.QueueElement_RefreshRelease{
					RefreshRelease: &pb.RefreshRelease{
						Iid:       req.GetInstanceId(),
						Intention: "Manual Update",
					}}},
		})

		if err != nil {
			return nil, err
		}
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			Force:   true,
			RunDate: time.Now().UnixNano(),
			Auth:    user.GetAuth().GetToken(),
			Entry: &pb.QueueElement_RefreshState{
				RefreshState: &pb.RefreshState{
					Iid: req.GetInstanceId(),
				}}},
	})

	return &pb.RefreshRecordResponse{}, err
}
