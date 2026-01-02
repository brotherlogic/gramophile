package server

import (
	"context"
	"log"
	"sort"

	"github.com/brotherlogic/gramophile/classification"
	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func reverse(rs []*pb.Record) []*pb.Record {
	for i := 0; i < len(rs)/2; i++ {
		j := len(rs) - i - 1
		rs[i], rs[j] = rs[j], rs[i]
	}
	return rs
}

func (s *Server) applyListeningFilter(ctx context.Context, f *pb.ListenFilter, rs []*pb.Record) *pb.Record {
	switch f.GetOrder().GetOrdering() {
	case pb.Order_ORDER_ADDED_DATE:
		sort.SliceStable(rs, func(i, j int) bool {
			return rs[i].GetRelease().GetInstanceId() < rs[j].GetRelease().GetInstanceId()
		})
	}

	if f.GetOrder().GetReverse() {
		reverse(rs)
	}

	for _, r := range rs {
		if config.Filter(f.GetFilter(), r) {
			return r
		}
	}

	return nil
}

func (s *Server) GetRecord(ctx context.Context, req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
	u, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.getRecordInternal(ctx, u, req)
	if err != nil {
		return resp, err
	}

	// Get any sale data
	for _, r := range resp.GetRecords() {
		if r.GetRecord().GetSaleId() > 0 {
			sale, err := s.d.GetSale(ctx, u.GetUser().GetDiscogsUserId(), r.GetRecord().GetSaleId())
			if err == nil {
				r.SaleInfo = sale
			}
		}
	}

	if req.IncludeHistory {
		for _, r := range resp.GetRecords() {
			up, err := s.d.GetUpdates(ctx, u.GetUser().DiscogsUserId, r.GetRecord())
			if err != nil {
				return nil, err
			}
			r.Updates = up
		}
	}

	// Classifiy out the records
	for _, r := range resp.GetRecords() {
		r.Category = classification.Classify(ctx, r.GetRecord(), u.GetConfig().GetClassificationConfig(), u.GetConfig().GetOrganisationConfig(), s.d, u.GetUser().GetDiscogsUserId())
	}

	return resp, err
}

func (s *Server) getRecordsPurchasedBetween(ctx context.Context, u *pb.StoredUser, req *pb.GetRecordsPurchasedBetween) (*pb.GetRecordResponse, error) {
	rs, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	var recs []*pb.RecordResponse
	for _, r := range rs {
		rec, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return nil, err
		}
		recs = append(recs, &pb.RecordResponse{Record: rec})
	}

	return &pb.GetRecordResponse{Records: recs}, nil
}

func (s *Server) getRecordInternal(ctx context.Context, u *pb.StoredUser, req *pb.GetRecordRequest) (*pb.GetRecordResponse, error) {
	if req.GetGetRecordsPurchasedBetween() != nil {
		return s.getRecordsPurchasedBetween(ctx, u, req.GetGetRecordsPurchasedBetween())
	}

	if req.GetGetRecordWithId() != nil && req.GetGetRecordWithId().GetInstanceId() > 0 {
		r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), req.GetGetRecordWithId().GetInstanceId())
		log.Printf("DataB RETURN %v, %v", r, err)
		if err != nil {
			return nil, err
		}
		return &pb.GetRecordResponse{Records: []*pb.RecordResponse{{Record: r}}}, nil
	} else if req.GetGetRecordWithId().GetReleaseId() > 0 {
		var records []*pb.RecordResponse
		rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		for _, r := range rids {
			r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
			if err != nil {
				return nil, err
			}

			if r.GetRelease().GetId() == req.GetGetRecordWithId().GetReleaseId() {
				records = append(records, &pb.RecordResponse{Record: r})
			}
		}
		return &pb.GetRecordResponse{Records: records}, nil
	} else if req.GetGetRecordsMintUp() {
		var records []*pb.RecordResponse
		rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		for _, rid := range rids {
			r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), rid)
			if err != nil {
				return nil, err
			}
			if r.GetKeepStatus() == pb.KeepStatus_MINT_UP_KEEP {
				records = append(records, &pb.RecordResponse{Record: r})
			}
		}
		return &pb.GetRecordResponse{Records: records}, nil
	} else if req.GetGetRecordWithId().GetLabelId() > 0 {
		var records []*pb.RecordResponse
		rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		for _, r := range rids {
			r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
			if err != nil {
				return nil, err
			}

			for _, l := range r.GetRelease().GetLabels() {
				if l.GetId() == req.GetGetRecordWithId().GetLabelId() {
					records = append(records, &pb.RecordResponse{Record: r})
				}
				break
			}
		}
		return &pb.GetRecordResponse{Records: records}, nil
	}

	if req.GetGetRecordToListenTo() != nil && req.GetGetRecordToListenTo().GetFilter() != "" {
		var filter *pb.ListenFilter
		for _, f := range u.GetConfig().GetListenConfig().GetFilters() {
			if f.GetName() == req.GetGetRecordToListenTo().GetFilter() {
				filter = f
				break
			}
		}
		if filter == nil {
			return nil, status.Errorf(codes.NotFound, "Unable to find a listening filter with name %v", req.GetGetRecordToListenTo().GetFilter())
		}

		rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
		if err != nil {
			return nil, err
		}

		var records []*pb.Record
		for _, r := range rids {
			r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
			if err != nil {
				return nil, err
			}

			records = append(records, r)
		}

		ret := s.applyListeningFilter(ctx, filter, records)

		if ret == nil {
			return nil, status.Errorf(codes.NotFound, "Unable to locate record to listen to from %v", req.GetGetRecordToListenTo().GetFilter())
		}
		return &pb.GetRecordResponse{Records: []*pb.RecordResponse{{Record: ret}}}, nil
	}

	rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	for _, rec := range rids {
		r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), rec)
		if err != nil {
			return nil, err
		}

		if req.GetGetRecordToListenTo() != nil {
			return &pb.GetRecordResponse{Records: []*pb.RecordResponse{{Record: r}}}, nil
		}

		if len(r.GetIssues()) > 0 {
			return &pb.GetRecordResponse{Records: []*pb.RecordResponse{{Record: r}}}, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "Unable to locate record with an issue")
}
