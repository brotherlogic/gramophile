package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	conn, err := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	sconn, serr := grpc.Dial(os.Args[1], grpc.WithTransportCredentials(insecure.NewCredentials()))
	if serr != nil {
		log.Fatalf("Bad dial: %v", err)
	}

	client := pb.NewQueueServiceClient(conn)
	sclient := pb.NewGramophileServiceClient(sconn)

	switch os.Args[2] {
	case "users":
		users, err := sclient.GetUsers(ctx, &pb.GetUsersRequest{})
		if err != nil {
			log.Fatalf("Error getting users: %v", err)
		}
		fmt.Printf("%v users\n", len(users.GetUsers()))
	case "refresh":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshUser{RefreshUser: &pb.RefreshUserEntry{Auth: os.Args[3]}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "collection":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshCollection{RefreshCollection: &pb.RefreshCollectionEntry{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	}
}
