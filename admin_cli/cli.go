package main

import (
	"context"
	"log"
	"os"

	"google.golang.org/grpc"

	pb "github.com/brotherlogic/gramophile/proto"
)

func main() {
	conn, err := grpc.Dial(os.Args[1])
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	client := pb.NewQueueServiceClient(conn)

	switch os.Args[2] {
	case "refresh":
		client.Execute(context.Background(), &pb.ExecuteRequest{
			Element: &pb.QueueElement{Token: os.Args[2], Secret: os.Args[3], Entry: &pb.QueueElement_RefreshUser{RefreshUser: &pb.RefreshUserEntry{Auth: os.Args[4]}}},
		})
	}
}
