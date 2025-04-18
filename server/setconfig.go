package server

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/gramophile/config"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) SetConfig(ctx context.Context, req *pb.SetConfigRequest) (*pb.SetConfigResponse, error) {
	u, err := s.getUser(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to get user: %w", err)
	}

	log.Printf("Got user: %v", u.GetUser())
	log.Printf("Down to: %v", s.di.ForUser(u.GetUser()))

	fields, err := s.di.ForUser(u.GetUser()).GetFields(ctx)
	if err != nil {
		return nil, fmt.Errorf("unable to promote di to user (or get fields): %w", err)
	}

	log.Printf("got these fields: %v", fields)

	u.Config = req.GetConfig()
	folders, verr := config.ValidateConfig(ctx, u, fields, u)
	if verr != nil {
		return nil, fmt.Errorf("bad validate: %w", verr)
	}

	log.Printf("got folders: %v", folders)

	for _, folder := range folders {
		s.qc.Enqueue(ctx, &pb.EnqueueRequest{
			Element: &pb.QueueElement{
				RunDate:          time.Now().UnixNano(),
				Auth:             u.GetAuth().GetToken(),
				BackoffInSeconds: 60,
				Entry: &pb.QueueElement_AddFolderUpdate{
					AddFolderUpdate: &pb.AddFolderUpdate{FolderName: folder.GetName()},
				},
			}})
	}

	u.LastConfigUpdate = time.Now().UnixNano()

	log.Printf("Updated user: %v", u)

	if req.GetConfig().GetWantsConfig().GetMintUpWantList() {
		s.d.SaveWantlist(ctx, u.GetUser().GetDiscogsUserId(),
			&pb.Wantlist{
				Name: "mint_up_wantlist",
			})
	} else {
		s.d.DeleteWantlist(ctx, u.GetUser().GetDiscogsUserId(), "mint_up_wantlist")
	}

	if req.GetConfig().GetWantsConfig().GetDigitalWantsList() {
		s.d.SaveWantlist(ctx, u.GetUser().GetDiscogsUserId(),
			&pb.Wantlist{
				Name: "digital_wantlist",
			})
	} else {
		s.d.DeleteWantlist(ctx, u.GetUser().GetDiscogsUserId(), "digital_wantlist")
	}

	// Apply the config
	keys, err := s.d.GetRecords(ctx, u.GetUser().GetDiscogsUserId())
	if err != nil {
		return nil, fmt.Errorf("error getting records: %w", err)
	}
	for _, key := range keys {
		r, err := s.d.GetRecord(ctx, u.GetUser().GetDiscogsUserId(), key)
		if err != nil {
			return nil, fmt.Errorf("error getting record from key: %v -> %w", key, err)
		}

		err = config.Apply(u.Config, r)
		if err != nil {
			return nil, fmt.Errorf("unable to apply config: %w", err)
		}

		err = s.d.SaveRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return nil, fmt.Errorf("unable to save record: %w", err)
		}
	}

	_, err = s.qc.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:          time.Now().UnixNano(),
			Auth:             u.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_MoveRecords{
				MoveRecords: &pb.MoveRecords{},
			},
		}})
	if err != nil {
		return nil, fmt.Errorf("unable to enqueue: %w", err)
	}

	return &pb.SetConfigResponse{}, s.d.SaveUser(ctx, u)
}
