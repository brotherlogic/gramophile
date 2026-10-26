package validatorlogic

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/queuelogic"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

const SaleStalenessThreshold = 7 * 24 * time.Hour

func IsSalesSyncActive(user *pb.StoredUser) bool {
	return user.GetSaleSyncActiveTime() != 0 && time.Since(time.Unix(0, user.GetSaleSyncActiveTime())) < 24*time.Hour
}

func ValidateUser(ctx context.Context, user *pb.StoredUser, client pb.GramophileServiceClient, queue pb.QueueServiceClient, d db.Database, ghclient ghbclient.GithubridgeClient) error {
	log.Printf("User Refresh %v -> %v", user, time.Since(time.Unix(0, user.GetLastRefreshTime())))

	if user.GetUserToken() == "" && time.Since(time.Unix(0, user.GetLastRefreshTime())) > time.Hour*24*7 {
		client.DeleteUser(ctx, &pb.DeleteUserRequest{Id: user.GetAuth().GetToken()})
	} else {
		if time.Since(time.Unix(0, user.GetLastRefreshTime())) > time.Hour*24*7 {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention:        "From Validator",
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 10,
					Entry: &pb.QueueElement_RefreshUser{
						RefreshUser: &pb.RefreshUserEntry{Auth: user.GetAuth().GetToken()},
					},
				},
			})
			if err != nil {
				log.Printf("unable to enqueue: %v", err)
			}
		}

		log.Printf("Collection: %v", time.Since(time.Unix(0, user.GetLastCollectionCheck())))
		if time.Since(time.Unix(0, user.GetLastCollectionCheck())) > queuelogic.CollectionCheck {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention:        "From Validator",
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
				log.Printf("Unable to validate collection: %v", err)
			}
		}

		log.Printf("Collection: %v", time.Since(time.Unix(0, user.GetLastCollectionRefresh())))
		if time.Since(time.Unix(0, user.GetLastCollectionRefresh())) > queuelogic.CollectionRefresh {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 15,
					Intention:        fmt.Sprintf("refreshing collection - %v", time.Since(time.Unix(0, user.GetLastCollectionRefresh()))),
					Entry: &pb.QueueElement_RefreshCollectionEntry{
						RefreshCollectionEntry: &pb.RefreshCollectionEntry{Page: 1},
					},
				},
			})
			if err != nil {
				log.Printf("unable to enqueue: %v", err)
			}
		}

		if IsSalesSyncActive(user) {
			log.Printf("Sales sync active since %v ago, skipping RefreshSales and ReconcileSales", time.Since(time.Unix(0, user.GetSaleSyncActiveTime())))
		} else {
			enqueuedReconcileSales := false
			log.Printf("ReconcileSales: %v", time.Since(time.Unix(0, user.GetLastSaleReconcile())))
			if time.Since(time.Unix(0, user.GetLastSaleReconcile())) > queuelogic.SaleReconcile {
				_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						Intention:        "From Validator (ReconcileSales)",
						RunDate:          time.Now().UnixNano(),
						Auth:             user.GetAuth().GetToken(),
						BackoffInSeconds: 15,
						Entry: &pb.QueueElement_ReconcileSales{
							ReconcileSales: &pb.ReconcileSales{Page: 1},
						},
					},
				})
				if err != nil {
					log.Printf("unable to enqueue: %v", err)
					if status.Code(err) == codes.AlreadyExists {
						enqueuedReconcileSales = true
					}
				} else {
					user.SaleSyncActiveTime = time.Now().UnixNano()
					err = d.SaveUser(ctx, user)
					if err != nil {
						return fmt.Errorf("unable to save user: %w", err)
					}
					enqueuedReconcileSales = true
				}
			}

			if !enqueuedReconcileSales {
				log.Printf("Sales: %v", time.Since(time.Unix(0, user.GetLastSaleRefresh())))
				if time.Since(time.Unix(0, user.GetLastSaleRefresh())) > time.Hour*24 {
					_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
						Element: &pb.QueueElement{
							Intention:        "From Validator",
							RunDate:          time.Now().UnixNano(),
							Auth:             user.GetAuth().GetToken(),
							BackoffInSeconds: 15,
							Entry: &pb.QueueElement_RefreshSales{
								RefreshSales: &pb.RefreshSales{Page: 1},
							},
						},
					})
					if err != nil {
						log.Printf("unable to enqueue: %v", err)
					} else {
						user.SaleSyncActiveTime = time.Now().UnixNano()
						err = d.SaveUser(ctx, user)
						if err != nil {
							return fmt.Errorf("unable to save user: %w", err)
						}
					}
				}
			}
		}

		log.Printf("AdjustSales: %v", time.Since(time.Unix(0, user.GetLastSaleAdjust())))
		if time.Since(time.Unix(0, user.GetLastSaleAdjust())) > time.Hour {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention:        "From Validator (AdjustSales)",
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 15,
					Entry: &pb.QueueElement_AdjustSales{
						AdjustSales: &pb.AdjustSales{},
					},
				},
			})
			if err != nil {
				log.Printf("unable to enqueue: %v", err)
			}
		}

		checkSalesStaleness(ctx, user, d, ghclient)

		log.Printf("Wants: %v", time.Since(time.Unix(0, user.GetLastWantRefresh())))
		if time.Since(time.Unix(0, user.GetLastWantRefresh())) > time.Hour*24 {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention:        "From Validator",
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 15,
					Entry: &pb.QueueElement_SyncWants{
						SyncWants: &pb.SyncWants{Page: 1},
					},
				},
			})
			if err != nil {
				log.Printf("unable to enqueue: %v", err)
			}
		}

		log.Printf("Orders: %v", time.Since(time.Unix(0, user.GetLastOrderSync())))
		if time.Since(time.Unix(0, user.GetLastOrderSync())) > time.Hour {
			_, err := queue.Enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					Intention:        "From Validator",
					RunDate:          time.Now().UnixNano(),
					Auth:             user.GetAuth().GetToken(),
					BackoffInSeconds: 15,
					Entry: &pb.QueueElement_SyncOrders{
						SyncOrders: &pb.SyncOrders{
							Page:   1,
							SyncId: time.Now().UnixNano(),
						},
					},
				},
			})
			if err != nil {
				log.Printf("unable to enqueue: %v", err)
			}
		}

		log.Printf("Running print loop")
		err := runPrintLoop(ctx, d, user)
		if err != nil {
			return fmt.Errorf("unable to run print loop: %w", err)
		}

		log.Printf("Running mint printer")
		err = runMintPrinter(ctx, d, user)
		if err != nil {
			return fmt.Errorf("unable to run mint printer: %w", err)
		}
	}

	return nil
}

func checkSalesStaleness(ctx context.Context, user *pb.StoredUser, d db.Database, ghclient ghbclient.GithubridgeClient) {
	if user.GetUserToken() == "" || ghclient == nil {
		return
	}

	var reasons []string

	refreshAge := time.Since(time.Unix(0, user.GetLastSaleRefresh()))
	if refreshAge > SaleStalenessThreshold {
		reasons = append(reasons, fmt.Sprintf("- User-level `LastSaleRefresh` is stale: last refreshed %v ago (timestamp: %v)", refreshAge.Round(time.Second), user.GetLastSaleRefresh()))
	}

	adjustAge := time.Since(time.Unix(0, user.GetLastSaleAdjust()))
	if adjustAge > SaleStalenessThreshold {
		reasons = append(reasons, fmt.Sprintf("- User-level `LastSaleAdjust` is stale: last adjusted %v ago (timestamp: %v)", adjustAge.Round(time.Second), user.GetLastSaleAdjust()))
	}

	if user.GetConfig().GetSaleConfig().GetHandlePriceUpdates() == pb.Enabled_ENABLED_ENABLED && d != nil {
		sids, err := d.GetSales(ctx, user.GetUser().GetDiscogsUserId())
		if err != nil {
			log.Printf("unable to get sales for staleness check on user %v: %v", user.GetUser().GetDiscogsUserId(), err)
		} else {
			saleCfg := user.GetConfig().GetSaleConfig()
			for _, sid := range sids {
				sale, err := d.GetSale(ctx, user.GetUser().GetDiscogsUserId(), sid)
				if err != nil {
					log.Printf("unable to read sale %v for staleness check on user %v: %v", sid, user.GetUser().GetDiscogsUserId(), err)
					continue
				}
				if shouldUpdate, updateTypeStr := shouldSaleHaveUpdatedWithinWeek(sale, saleCfg); shouldUpdate {
					priceAge := time.Since(time.Unix(0, sale.GetLastPriceUpdate()))
					if priceAge > SaleStalenessThreshold {
						reasons = append(reasons, fmt.Sprintf("- Sale `%v` (release `%v`, update_type: `%v`, current_price: `%v`) has not updated in %v (last_price_update: %v)",
							sale.GetSaleId(), sale.GetReleaseId(), updateTypeStr, sale.GetCurrentPrice().GetValue(), priceAge.Round(time.Second), sale.GetLastPriceUpdate()))
					}
				}
			}
		}
	}

	if len(reasons) > 0 {
		reportSalesStaleness(ctx, user, reasons, ghclient)
	}
}

func shouldSaleHaveUpdatedWithinWeek(sale *pb.SaleInfo, c *pb.SaleConfig) (bool, string) {
	if sale == nil || c == nil {
		return false, ""
	}
	if c.GetHandlePriceUpdates() != pb.Enabled_ENABLED_ENABLED {
		return false, ""
	}
	if sale.GetSaleState() != pbd.SaleStatus_FOR_SALE {
		return false, ""
	}

	updateType := c.GetUpdateType()
	if sale.GetSaleUpdateOverride() != pb.SaleUpdateType_SALE_UPDATE_UNKNOWN {
		updateType = sale.GetSaleUpdateOverride()
	}
	if updateType == pb.SaleUpdateType_NO_SALE_UPDATE || updateType == pb.SaleUpdateType_SALE_UPDATE_UNKNOWN {
		return false, ""
	}

	if time.Duration(c.GetUpdateFrequencySeconds())*time.Second >= SaleStalenessThreshold {
		return false, ""
	}

	switch updateType {
	case pb.SaleUpdateType_REDUCE_TO_MEDIAN:
		if sale.GetTimeAtMedian() > 0 || (sale.GetMedianPrice().GetValue() > 0 && sale.GetCurrentPrice().GetValue() <= sale.GetMedianPrice().GetValue()) {
			return false, ""
		}
	case pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW:
		lowerBound := c.GetLowerBound()
		if c.GetLowerBoundStrategy() == pb.LowerBoundStrategy_DISCOGS_LOW {
			lowerBound = sale.GetLowPrice().GetValue()
		}
		if sale.GetTimeAtLow() > 0 || (lowerBound > 0 && sale.GetCurrentPrice().GetValue() <= lowerBound) {
			return false, ""
		}
		atOrPostMedian := sale.GetTimeAtMedian() > 0 || (sale.GetMedianPrice().GetValue() > 0 && sale.GetCurrentPrice().GetValue() <= sale.GetMedianPrice().GetValue())
		if atOrPostMedian {
			if c.GetTimeToMedianDays() == 0 && c.GetPostMedianReduction() <= 0 {
				return false, ""
			}
			if sale.GetTimeAtMedian() > 0 && time.Since(time.Unix(0, sale.GetTimeAtMedian())) <= time.Duration(c.GetPostMedianTime())*time.Second {
				return false, ""
			}
			if c.GetTimeToMedianDays() == 0 && time.Duration(c.GetPostMedianReductionFrequency())*time.Second >= SaleStalenessThreshold {
				return false, ""
			}
		}
	case pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW_AND_THEN_STALE:
		if sale.GetTimeAtStale() > 0 || (c.GetStaleBound() > 0 && sale.GetCurrentPrice().GetValue() <= c.GetStaleBound()) {
			return false, ""
		}
		lowerBound := c.GetLowerBound()
		if c.GetLowerBoundStrategy() == pb.LowerBoundStrategy_DISCOGS_LOW {
			lowerBound = sale.GetLowPrice().GetValue()
		}
		atOrPostLow := sale.GetTimeAtLow() > 0 || (lowerBound > 0 && sale.GetCurrentPrice().GetValue() <= lowerBound)
		if atOrPostLow {
			if c.GetPostLowReduction() <= 0 {
				return false, ""
			}
			if sale.GetTimeAtLow() > 0 && time.Since(time.Unix(0, sale.GetTimeAtLow())) <= time.Duration(c.GetTimeToStaleDays())*24*time.Hour {
				return false, ""
			}
			if time.Duration(c.GetPostLowReductionFrequencySeconds())*time.Second >= SaleStalenessThreshold {
				return false, ""
			}
		} else {
			atOrPostMedian := sale.GetTimeAtMedian() > 0 || (sale.GetMedianPrice().GetValue() > 0 && sale.GetCurrentPrice().GetValue() <= sale.GetMedianPrice().GetValue())
			if atOrPostMedian {
				if c.GetTimeToMedianDays() == 0 && c.GetPostMedianReduction() <= 0 {
					return false, ""
				}
				if sale.GetTimeAtMedian() > 0 && time.Since(time.Unix(0, sale.GetTimeAtMedian())) <= time.Duration(c.GetPostMedianTime())*time.Second {
					return false, ""
				}
				if c.GetTimeToMedianDays() == 0 && time.Duration(c.GetPostMedianReductionFrequency())*time.Second >= SaleStalenessThreshold {
					return false, ""
				}
			}
		}
	}

	return true, updateType.String()
}

func reportSalesStaleness(ctx context.Context, user *pb.StoredUser, reasons []string, ghclient ghbclient.GithubridgeClient) {
	userID := fmt.Sprintf("%v", user.GetUser().GetDiscogsUserId())
	if user.GetUser().GetDiscogsUserId() == 0 {
		userID = user.GetAuth().GetToken()
	}

	title := fmt.Sprintf("Sales staleness detected for user %v", userID)

	resp, err := ghclient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		log.Printf("unable to list issues for sales staleness deduplication check on user %v: %v", userID, err)
	} else {
		for _, issue := range resp.GetIssues() {
			if issue.GetTitle() == title && issue.GetState() != ghbpb.IssueState_ISSUE_STATE_CLOSED {
				log.Printf("open issue already exists for sales staleness on user %v: %v (id: %v), skipping creation", userID, title, issue.GetId())
				return
			}
		}
	}

	body := fmt.Sprintf("Sales staleness detected during validation for user %v:\n\n%s", userID, strings.Join(reasons, "\n"))

	_, err = ghclient.CreateIssue(ctx, &ghbpb.CreateIssueRequest{
		User:  "brotherlogic",
		Repo:  "gramophile",
		Title: title,
		Body:  body,
	})
	if err != nil {
		log.Printf("unable to create github issue for sales staleness on user %v: %v", userID, err)
	}
}

func RunValidationLoop(ctx context.Context) error {
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
		return fmt.Errorf("unable to get users: %v", err)
	}

	d := db.NewDatabase(ctx)

	ghclient, err := ghbclient.GetClientInternal()
	if err != nil {
		log.Printf("unable to get githubridge client: %v", err)
	}

	for _, user := range users.GetUsers() {
		err := ValidateUser(ctx, user, client, queue, d, ghclient)
		if err != nil {
			log.Printf("unable to validate user %v: %v", user.GetAuth().GetToken(), err)
		}
	}

	return nil
}
