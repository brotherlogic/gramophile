package server

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) LocateRecord(ctx context.Context, req *pb.LocateRecordRequest) (*pb.LocateRecordResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "not implemented")
}
