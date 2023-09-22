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
		return nil, err
	}

	log.Printf("Got user: %v", u.GetUser())
	log.Printf("Down to: %v", s.di.ForUser(u.GetUser()))

	fields, err := s.di.ForUser(u.GetUser()).GetFields(ctx)
	if err != nil {
		return nil, err
	}

	log.Printf("got fields: %v", fields)

	folders, moves, verr := config.ValidateConfig(ctx, u, fields, req.GetConfig())
	if verr != nil {
		return nil, fmt.Errorf("bad validate: %v", verr)
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

	u.Moves = append(u.Moves, moves...)
	u.Config = req.GetConfig()
	u.LastConfigUpdate = time.Now().Unix()

	log.Printf("Updated user: %v", u)

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
			return nil, err
		}

		err = s.d.SaveRecord(ctx, u.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return nil, err
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
		return nil, err
	}

	return &pb.SetConfigResponse{}, s.d.SaveUser(ctx, u)
}
