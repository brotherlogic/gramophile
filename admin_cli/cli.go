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
		a, b := client.Execute(context.Background(), &pb.ExecuteRequest{
			Element: &pb.QueueElement{Token: os.Args[3], Secret: os.Args[4], Entry: &pb.QueueElement_RefreshUser{RefreshUser: &pb.RefreshUserEntry{Auth: os.Args[4]}}},
		})
		fmt.Printf("%v and %v\n", a, b)
	}
}
