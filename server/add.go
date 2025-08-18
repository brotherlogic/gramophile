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

	if user.GetConfig().GetAddConfig().GetAdds() == pb.Enabled_ENABLED_ENABLED {
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
	err = s.d.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), &pb.Record{
		Release: &pbd.Release{Id: req.GetId(), InstanceId: iid},
	})
	if err != nil {
		return nil, err
	}

	// Lets update the want status to see if we bought any wants
	wants, err := s.d.GetWants(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, err
	}

	for _, want := range wants {
		if want.GetId() == req.GetId() {
			log.Printf("FOUND WANT %v AND SETTING PURCHASED", want)
			want.IntendedState = pb.WantState_IN_TRANSIT
			err = s.d.SaveWant(ctx, user.GetUser().GetDiscogsUserId(), want, "Saving because purchased")
			if err != nil {
				return nil, err
			}
			// Trigger a want update to ensure we capture wantlist changes
			s.qc.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate: time.Now().UnixNano(),
					Auth:    user.GetAuth().GetToken(),
					Entry: &pb.QueueElement_RefreshWant{
						RefreshWant: &pb.RefreshWant{Want: want},
					},
				},
			})
		}
	}

	// Trigger a wantlist update to ensure we capture wantlist changes
	s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate: time.Now().UnixNano(),
			Auth:    user.GetAuth().GetToken(),
			Entry: &pb.QueueElement_RefreshWantlists{
				RefreshWantlists: &pb.RefreshWantlists{},
			},
		},
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
