package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

func main() {
	conn, err := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	client := pb.NewQueueServiceClient(conn)

	switch os.Args[2] {
	case "refresh":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshUser{RefreshUser: &pb.RefreshUserEntry{Auth: os.Args[3]}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "collection":
		a, b := client.Execute(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshCollection{RefreshCollection: &pb.RefreshCollectionEntry{Page: 4}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	}
}
