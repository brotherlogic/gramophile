package main

import (
	"context"
	"log"
	"time"

	"google.golang.org/grpc"

	pb "github.com/brotherlogic/gramophile/proto"
)

func validateUsers(ctx context.Context) error {
	conn, err := grpc.Dial("gramophile.gramophile:8083", grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	qconn, err := grpc.Dial("gramophile:gramophile-queue:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pb.NewGramophileServiceClient(conn)
	queue := pb.NewQueueServiceClient(qconn)
	users, err := client.GetUsers(ctx, &pb.GetUsersRequest{})
	if err != nil {
		return err
	}

	for _, user := range users.GetUsers() {
		log.Printf("%v -> %v", user, time.Since(time.Unix(user.GetLastRefreshTime(), 0)))
		if user.GetUserToken() == "" && time.Since(time.Unix(user.GetLastRefreshTime(), 0)) > time.Hour*24*7 {
			client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.GetAuth().GetToken()})
		} else {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().Unix(),
					Token:            user.GetUserToken(),
					Secret:           user.GetUserSecret(),
					BackoffInSeconds: 10,
					Entry: &pb.QueueElement_RefreshUser{
						RefreshUser: &pb.RefreshUserEntry{Auth: user.GetAuth().GetToken()},
					},
				},
			})
			if err != nil {
				return err
			}

			log.Printf("Collection: %v", time.Since(time.Unix(user.GetLastRefreshTime(), 0)))
			if time.Since(time.Unix(user.GetLastCollectionRefresh(), 0)) > time.Hour*24*7 {
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().Unix(),
						Token:            user.GetUserToken(),
						Secret:           user.GetUserSecret(),
						BackoffInSeconds: 10,
						Entry: &pb.QueueElement_RefreshCollection{
							RefreshCollection: &pb.RefreshCollectionEntry{Page: 1},
						},
					},
				})
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func main() {
	log.Printf("Starting validator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	err := validateUsers(ctx)
	if err != nil {
		log.Fatalf("Cannot validate users: %v", err)
	}
}
