package server

import (
	"context"
	"fmt"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (s *Server) AddRecord(ctx context.Context, req *pb.AddRecordRequest) (*pb.AddRecordResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}

	if user.GetConfig().GetAddConfig().GetAllowAdds() == pb.Mandate_NONE {
		return nil, status.Errorf(codes.FailedPrecondition, "Adding is disabled")
	}

	// Resolve the default folder
	defaultFolder := int64(0)
	for _, folder := range user.GetFolders() {
		if folder.GetName() == user.GetConfig().GetAddConfig().GetDefaultFolder() {
			defaultFolder = int64(folder.GetId())
		}
	}

	if defaultFolder == 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "Your default folder (%v) does not exist", user.GetConfig().GetAddConfig().GetDefaultFolder())
	}

	iid := int64(0)
	attempts := 0
	for iid == 0 {
		// Cheap Backoff
		time.Sleep(time.Second * time.Duration(attempts))

		iid, err = s.di.AddRelease(ctx, req.GetId(), defaultFolder)
		if err != nil && status.Code(err) != codes.ResourceExhausted {
			return nil, fmt.Errorf("unable to add record: %w", err)
		}
		attempts++
	}

	if iid == 0 {
		return nil, fmt.Errorf("Permanent failure trying to add record, last error was: %v", err)
	}

	return &pb.AddRecordResponse{InstanceId: iid}, nil
}
