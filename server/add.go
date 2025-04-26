package server

import (
	"context"
	"fmt"
	"log"
	"time"

	pbd "github.com/brotherlogic/discogs/proto"
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

		iid, err = s.di.ForUser(user.GetUser()).AddRelease(ctx, req.GetId(), defaultFolder)
		if err != nil && status.Code(err) != codes.ResourceExhausted {
			return nil, fmt.Errorf("unable to add record: %w", err)
		}
		log.Printf("Got result %v and %v", iid, err)
		attempts++
	}

	if iid == 0 {
		return nil, fmt.Errorf("Permanent failure trying to add record, last error was: %v", err)
	}

	// Save the new record and enqueue updates for price and location
	s.d.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), &pb.Record{
		Release: &pbd.Release{InstanceId: iid},
	})

	s.SetIntent(ctx, &pb.SetIntentRequest{
		InstanceId: iid,
		Intent: &pb.Intent{
			PurchaseLocation: req.GetLocation(),
			PurchasePrice:    req.GetPrice(),
		},
	})

	return &pb.AddRecordResponse{InstanceId: iid}, nil
}
