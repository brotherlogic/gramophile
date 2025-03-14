package server

import (
	"context"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetSale(ctx context.Context, req *pb.GetSaleRequest) (*pb.GetSaleResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	if req.GetMinMedian() > 0 {
		var ret []*pb.SaleInfo
		sales, err := s.d.GetSales(ctx, user.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		for _, sid := range sales {
			sale, err := s.d.GetSale(ctx, user.GetUser().GetDiscogsUserId(), sid)
			if err != nil {
				return nil, err
			}
			if time.Since(time.Unix(0, sale.GetTimeAtMedian())) > time.Second*time.Duration(req.GetMinMedian()) {
				ret = append(ret, sale)
			}
		}

		return &pb.GetSaleResponse{Sales: ret}, nil
	}

	if req.GetMinMedian() < 0 {
		var ret []*pb.SaleInfo
		sales, err := s.d.GetSales(ctx, user.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		for _, sid := range sales {
			sale, err := s.d.GetSale(ctx, user.GetUser().GetDiscogsUserId(), sid)
			if err != nil {
				return nil, err
			}
			ret = append(ret, sale)
		}

		return &pb.GetSaleResponse{Sales: ret}, nil
	}

	saleinfo, err := s.d.GetSale(ctx, user.GetUser().GetDiscogsUserId(), req.GetId())
	if err != nil {
		return nil, err
	}

	return &pb.GetSaleResponse{Sales: []*pb.SaleInfo{saleinfo}}, nil
}
