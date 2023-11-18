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

	qconn, err := grpc.Dial("gramophile-queue.gramophile:8080", grpc.WithInsecure())
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
		log.Printf("User Refresh %v -> %v", user, time.Since(time.Unix(user.GetLastRefreshTime(), 0)))

		if user.GetUserToken() == "" && time.Since(time.Unix(user.GetLastRefreshTime(), 0)) > time.Hour*24*7 {
			client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.GetAuth().GetToken()})
		} else {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 10,
					Entry: &pb.QueueElement_RefreshUser{
						RefreshUser: &pb.RefreshUserEntry{Auth: user.GetAuth().GetToken()},
					},
				},
			})
			if err != nil {
				return err
			}

			_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 10,
					Entry:            &pb.QueueElement_RefreshUpdates{},
				},
			})
			if err != nil {
				return err
			}

			log.Printf("Collection: %v", time.Since(time.Unix(user.GetLastRefreshTime(), 0)))
			if time.Since(time.Unix(user.GetLastCollectionRefresh(), 0)) > time.Hour*24 {
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_RefreshCollection{
							RefreshCollection: &pb.RefreshCollection{},
						},
					},
				})
				if err != nil {
					return err
				}
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_RefreshCollectionEntry{
							RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
						},
					},
				})
				if err != nil {
					return err
				}
			}

			log.Printf("Sales: %v", time.Since(time.Unix(0, user.GetLastSaleRefresh())))
			if time.Since(time.Unix(0, user.GetLastSaleRefresh())) > time.Minute*50 {
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_RefreshSales{
							RefreshSales: &pb.RefreshSales{Page: 1},
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
