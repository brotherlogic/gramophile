package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
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
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshCollectionEntry{RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "clean":
		_, err := sclient.Clean(ctx, &pb.CleanRequest{})
		if err != nil {
			log.Fatalf("Error in clean: %v", err)
		}
	case "list":
		items, err := client.List(context.Background(), &pb.ListRequest{})
		if err != nil {
			log.Fatalf("Bad list: %v", err)
		}
		for _, item := range items.GetElements() {
			fmt.Printf("%v\n", item)
		}
	case "syncsales":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "refresh_release":
		iid, err := strconv.ParseInt(os.Args[4], 10, 64)
		if err != nil {
			log.Fatalf("Unable to parse %v -> %v", os.Args[4], err)
		}
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshRelease{RefreshRelease: &pb.RefreshRelease{Iid: iid}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	case "adjustsales":
		a, b := client.Enqueue(context.Background(), &pb.EnqueueRequest{
			Element: &pb.QueueElement{Auth: os.Args[3], Entry: &pb.QueueElement_RefreshSales{RefreshSales: &pb.RefreshSales{Page: 1}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	}
}
