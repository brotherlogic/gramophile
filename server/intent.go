package server

import (
	"context"
	"log"
	"time"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func (s *Server) SetIntent(ctx context.Context, req *pb.SetIntentRequest) (*pb.SetIntentResponse, error) {
	user, err := s.getUser(ctx)
	if err != nil {
		return nil, err
	}
	exint, err := s.d.GetIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId())
	if err != nil {
		if status.Code(err) == codes.NotFound {
			exint = &pb.Intent{}
		} else {
			return nil, err
		}
	}

	// Merge in the proto def
	proto.Merge(exint, req.GetIntent())

	err = s.d.SaveIntent(ctx, user.GetUser().GetDiscogsUserId(), req.GetInstanceId(), exint)
	if err != nil {
		return nil, err
	}

	conn, err := grpc.Dial("gramophile-queue.gramophile:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	client := pb.NewQueueServiceClient(conn)
	_, err = client.Enqueue(ctx, &pb.EnqueueRequest{
		Element: &pb.QueueElement{
			RunDate:          time.Now().UnixNano(),
			Auth:             user.GetAuth().GetToken(),
			BackoffInSeconds: 60,
			Entry: &pb.QueueElement_RefreshIntents{
				RefreshIntents: &pb.RefreshIntents{InstanceId: req.GetInstanceId()},
			},
		},
	})

	log.Printf("Saved Intent")

	return &pb.SetIntentResponse{}, err
}
