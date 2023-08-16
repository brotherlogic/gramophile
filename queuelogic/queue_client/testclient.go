package queue_client

import (
	"context"

	pb "github.com/brotherlogic/gramophile/proto"
)

type TestClient struct {
	list []*pb.QueueElement
}

func GetTestClient() QueueClient {
	return &TestClient{list: make([]*pb.QueueElement, 0)}
}

func (c *TestClient) Enqueue(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
	c.list = append(c.list, req.GetElement())
	return &pb.EnqueueResponse{}, nil
}
