package server

import (
	"context"
	"fmt"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) validateIntent(ctx context.Context, user *pb.StoredUser, i *pb.Intent) error {
	if i.GetGoalFolder() != "" {
		found := false
		for _, folder := range user.GetFolders() {
			if folder.GetName() == i.GetGoalFolder() {
				found = true
			}
		}

		if !found {
			return status.Errorf(codes.FailedPrecondition, "%v is not in the list of current user folders", i.GetGoalFolder())
		}
	}

	if i.GetSleeve() != "" {
		found := false
		for _, sleeve := range user.GetConfig().GetSleeveConfig().GetAllowedSleeves() {
			if sleeve.GetName() == i.GetSleeve() {
				found = true
			}
		}

		if !found {
			return status.Errorf(codes.FailedPrecondition, "%v is not in the list of allowed sleeves", i.GetSleeve())
		}
	}

	return nil
}

func (s *Server) SetIntent(ctx context.Context, req *pb.SetIntentRequest) (*pb.SetIntentResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("Error getting user: %w", err)
	}

	// Check that this record at least exists
	_, err = s.d.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		return nil, fmt.Errorf("error getting record: %w", err)
	}

	if req.GetIntent().GetKeep() == pb.KeepStatus_MINT_UP_KEEP && len(req.GetIntent().GetMintIds()) == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "You need to specify mint ids for this keep")
	}

	exint, err := s.d.GetIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			exint = &pb.Intent{}
		} else {
			return nil, fmt.Errorf("error getting intents: %w", err)
		}
	}

	// Merge in the proto def
	proto.Merge(exint, req.GetIntent())

	// Validate that the intent is legit
	err = s.validateIntent(ctx, user, exint)
	if err != nil {
		return nil, err
	}

	log.Printf("Saving intent: %v -> %v", exint, user)

	// Clear the score if we've moved into the listening pile
	if req.GetIntent().GetNewFolder() == 812802 || req.GetIntent().GetNewFolder() == 7651472 || req.GetIntent().GetNewFolder() == 7664293 || req.GetIntent().GetNewFolder() == 7665013 {
		exint.NewScore = -1
	}

	err = s.d.SaveIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId(), exint)
	if err != nil {
		return nil, fmt.Errorf("error saving intent: %w", err)
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:          time.Now().UnixNano(), // We want intents to run before anything else
			Priority:         pb.QueueElement_PRIORITY_HIGH,
			Auth:             user.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_RefreshIntents{
				RefreshIntents: &pb.RefreshIntents{InstanceId: req.GetInstanceId()},
			},
		},
	})

	log.Printf("Saved Intent")
	if user.GetUser().GetDiscogsUserId() == 150295 {
		nerr := s.updateRecord(ctx, int32(req.GetInstanceId()))
		if nerr != nil {
			log.Printf("Error on record update: %v", nerr)
		}
	} else {
		log.Printf("Skipping: %v", user.GetUser().GetDiscogsUserId())
	}

	return &pb.SetIntentResponse{}, err
}
