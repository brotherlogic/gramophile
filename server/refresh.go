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

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: time.Now().UnixNano(),
			Auth:    user.GetAuth().GetToken(),
			Entry: &pb.QueueElement_RefreshRelease{
				RefreshRelease: &pb.RefreshRelease{
					Iid:       req.GetInstanceId(),
					Intention: "Manual Update",
				}}},
	})

	return &pb.RefreshRecordResponse{}, err
}
