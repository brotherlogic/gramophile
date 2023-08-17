package queue_client

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc"
)

type QueueClient interface {
	Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error)
}

type qClient struct {
	qClient pb.QueueServiceClient
}

func GetClient() (QueueClient, error) {
	conn, err := grpc.Dial("rstore.rstore:8080", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return &qClient{qClient: pb.NewQueueServiceClient(conn)}, nil
}

func (q *qClient) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	return q.qClient.Enqueue(ctx, req)
}
