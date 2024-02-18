package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetSale(ctx context.Context, req *pb.GetSaleRequest) (*pb.GetSaleResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	saleinfo, err := s.d.GetSale(ctx, user.GetUser().GetDiscogsUserId(), req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.GetSaleResponse{Sale: saleinfo}, nil
}
