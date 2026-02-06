package server

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) RefreshRecord(ctx context.Context, req *pb.RefreshRecordRequest) (*pb.RefreshRecordResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting user: %w", err)
	}

	if req.GetInstanceId() == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "Cannot refresh zeroth element")
	}

	if !req.JustState {
		_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				Force:     true,
				RunDate:   time.Now().UnixNano(),
				Intention: "Running from RefreshRecord",
				Auth:      user.GetAuth().GetToken(),
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
			Force:     true,
			RunDate:   time.Now().UnixNano(),
			Intention: "Running from RefreshRecord",
			Auth:      user.GetAuth().GetToken(),
			Entry: &pb.QueueElement_RefreshState{
				RefreshState: &pb.RefreshState{
					Iid: req.GetInstanceId(),
				}}},
	})

	record, err := s.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		return nil, err
	}

	return &pb.RefreshRecordResponse{SaleId: record.GetSaleId(), HighPrice: record.GetHighPrice().GetValue()}, err
}
