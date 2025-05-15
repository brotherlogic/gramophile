package server

import (
	"context"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func (s *Server) AddSale(ctx context.Context, req *pb.AddSaleRequest) (*pb.AddSaleResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	// Validate that we own a record with this release id and that one of them doesn't have a sale id
	records, err := s.GetRecord(ctx, &pb.GetRecordRequest{
		Request: &pb.GetRecordRequest_GetRecordWithId{
			GetRecordWithId: &pb.GetRecordWithId{ReleaseId: req.GetParams().GetReleaseId()},
		},
	})
	if err != nil {
		return nil, err
	}

	var foundRecord *pb.Record
	for _, record := range records.GetRecords() {
		if record.GetSaleInfo().GetSaleId() == 0 {
			foundRecord = record.GetRecord()
			break
		}
	}

	if foundRecord == nil {
		return nil, status.Errorf(codes.FailedPrecondition, "You cannot sell a record you do not own")
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:  time.Now().UnixNano(),
			Auth:     user.GetAuth().GetToken(),
			Priority: pb.QueueElement_PRIORITY_LOW,
			Entry: &pb.QueueElement_AddSale{
				AddSale: &pb.AddSale{
					InstanceId: foundRecord.GetRelease().GetInstanceId(),
					SaleParams: req.GetParams()},
			},
		},
	})
	return &pb.AddSaleResponse{}, err
}
