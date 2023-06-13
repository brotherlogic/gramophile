package server

import (
	"context"
	"math/rand"
	"sort"
	"time"

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

	rids, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(rids), func(i, j int) { rids[i], rids[j] = rids[j], rids[i] })

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
		return &pb.GetRecordResponse{Record: ret}, nil
	}

	for _, rec := range rids {
		r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), rec)
		if err != nil {
			return nil, err
		}

		if req.GetGetRecordToListenTo() != nil {
			return &pb.GetRecordResponse{Record: r}, nil
		}

		if len(r.GetIssues()) > 0 {
			return &pb.GetRecordResponse{Record: r}, nil
		}
	}

	return nil, status.Errorf(codes.NotFound, "Unable to locate record with an issue")
}
