package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/brotherlogic/gramophile/proto"
)

func runValidationLoop(ctx context.Context) error {
	conn, err := grpc.Dial("gramophile.gramophile:8083", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	qconn, err := grpc.Dial("gramophile-queue.gramophile:8080", grpc.WithTransportCredentials(insecure.NewCredentials()))
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
		log.Printf("User Refresh %v -> %v", user, time.Since(time.Unix(0, user.GetLastRefreshTime())))

		if user.GetUserToken() == "" && time.Since(time.Unix(0, user.GetLastRefreshTime())) > time.Hour*24*7 {
			client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.GetAuth().GetToken()})
		} else {
			if time.Since(time.Unix(0, user.GetLastRefreshTime())) > time.Hour*24*7 {
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
			}

			log.Printf("Collection: %v", time.Since(time.Unix(0, user.GetLastRefreshTime())))
			if time.Since(time.Unix(0, user.GetLastCollectionRefresh())) > time.Hour*8 {
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_RefreshCollection{
							RefreshCollection: &pb.RefreshCollection{
								Intention: fmt.Sprintf("from-validator-%v", time.Since(time.Unix(0, user.GetLastCollectionRefresh()))),
							},
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
			if time.Since(time.Unix(0, user.GetLastSaleRefresh())) > time.Hour*24 {
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

			log.Printf("Wants: %v", time.Since(time.Unix(0, user.GetLastWantRefresh())))
			if time.Since(time.Unix(0, user.GetLastWantRefresh())) > time.Hour*24 {
				_, err = queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_SyncWants{
							SyncWants: &pb.SyncWants{Page: 1},
						},
					},
				})
				if err != nil {
					return err
				}
			}

			log.Printf("Running print loop")
			err = runPrintLoop(ctx, user)
			if err != nil {
				return err
			}

			log.Printf("Running mint printer")
			err = runMintPrinter(ctx, user)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func main() {
	log.Printf("Starting validator")
	ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
	defer cancel()

	ts := time.Now()
	err := runValidationLoop(ctx)
	log.Printf("Completing validation in %v", time.Since(ts))
	if err != nil {
		log.Fatalf("Cannot run validation loop: %v", err)
	}
}
