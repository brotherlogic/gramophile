package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/brotherlogic/gramophile/background"

	rstore_client "github.com/brotherlogic/rstore/client"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	pb "github.com/brotherlogic/gramophile/proto"
	rspb "github.com/brotherlogic/rstore/proto"
)

var (
	QUEUE_SUFFIX = "gramophile/taskqueue/"
)

type Queue struct {
	rstore     *rstore_client.RStoreClient
	Background *background.BackgroundRunner
}

func (q *Queue) run() {
	for {
		ctx := context.Background()
		entry, err := q.getNextEntry(ctx)
		if err == nil {
			_, err = q.Execute(ctx, &pb.ExecuteRequest{Element: entry})
		}

		// Back off on any type of error
		if err == nil {
			q.delete(ctx, entry)
		} else {
			time.Sleep(time.Second * time.Duration(entry.GetBackoffInSeconds()))
		}
	}
}

func (q *Queue) Execute(ctx context.Context, req *pb.ExecuteRequest) (*pb.ExecuteResponse, error) {
	return &pb.ExecuteResponse{}, q.ExecuteInternal(ctx, req.GetElement())
}

func (q *Queue) ExecuteInternal(ctx context.Context, entry *pb.QueueElement) error {
	switch entry.Entry.(type) {
	case *pb.QueueElement_RefreshUser:
		return q.Background.RefreshUser(ctx, entry.GetRefreshUser().GetAuth(), entry.GetToken(), entry.GetSecret())
	}

	return status.Errorf(codes.NotFound, "Unable to handle %v", entry)
}

func (q *Queue) delete(ctx context.Context, entry *pb.QueueElement) error {
	_, err := q.rstore.Delete(ctx, &rspb.DeleteRequest{Key: fmt.Sprintf("%v/%v", QUEUE_SUFFIX, entry.GetRunDate())})
	return err
}

func (q *Queue) getNextEntry(ctx context.Context) (*pb.QueueElement, error) {
	keys, err := q.rstore.GetKeys(ctx, &rspb.GetKeysRequest{Suffix: QUEUE_SUFFIX})
	if err != nil {
		return nil, err
	}

	data, err := q.rstore.Read(ctx, &rspb.ReadRequest{Key: keys.GetKeys()[0]})
	if err != nil {
		return nil, err
	}

	entry := &pb.QueueElement{}
	err = proto.Unmarshal(data.GetValue().GetValue(), entry)
	return entry, err
}
