package server

import (
	"context"
	"math/rand"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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
