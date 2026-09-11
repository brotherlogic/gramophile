package background

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	salesMarkedSold = promauto.NewCounter(prometheus.CounterOpts{
		Name: "gramophile_sales_marked_sold",
		Help: "Total sold items detected via order sync",
	})

	orderSyncTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_order_sync_total",
		Help: "Total order syncs executed",
	}, []string{"status"})

	orderSyncErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "gramophile_order_sync_errors",
		Help: "Total order sync errors",
	}, []string{"code"})
)

type syncOrdersHandler struct {
	b *BackgroundRunner
}

func (h *syncOrdersHandler) Execute(ctx context.Context, d discogs.Discogs, u *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	return h.b.ProcessSyncOrders(ctx, d, u, entry, enqueue)
}

func (h *syncOrdersHandler) Validate(ctx context.Context, db db.Database, entry *pb.QueueElement) error {
	return nil
}

func (h *syncOrdersHandler) GetDeduplicationKey(entry *pb.QueueElement) string {
	return ""
}

func (b *BackgroundRunner) SyncOrders(ctx context.Context, d discogs.Discogs, user *pb.StoredUser, page int32) (*pbd.Pagination, error) {
	var createdAfter time.Time
	if user.GetLastOrderSync() == 0 {
		createdAfter = time.Now().Add(-30 * 24 * time.Hour)
	} else {
		createdAfter = time.Unix(0, user.GetLastOrderSync())
	}

	orders, pagination, err := d.ListOrders(ctx, createdAfter, page)
	if err != nil {
		return nil, err
	}

	now := time.Now().UnixNano()
	for _, order := range orders {
		if strings.HasPrefix(strings.ToLower(order.GetStatus()), "cancelled") {
			continue
		}

		soldDate := order.GetCreated()
		if soldDate < 1e12 && soldDate > 0 {
			soldDate = time.Unix(soldDate, 0).UnixNano()
		}

		for _, item := range order.GetItems() {
			sale, err := b.db.GetSale(ctx, d.GetUserId(), item.GetId())
			if err != nil {
				if status.Code(err) == codes.NotFound {
					newSale := &pb.SaleInfo{
						SaleId:          item.GetId(),
						ReleaseId:       item.GetReleaseId(),
						SaleState:       pbd.SaleStatus_SOLD,
						Condition:       item.GetCondition(),
						SleeveCondition: item.GetSleeveCondition(),
						CurrentPrice:    item.GetPrice(),
						SoldDate:        soldDate,
						TimeCreated:     now,
						TimeRefreshed:   now,
						LastPriceUpdate: now,
					}
					err = b.db.SaveSale(ctx, d.GetUserId(), newSale)
					if err != nil {
						return nil, err
					}
					salesMarkedSold.Inc()
				} else {
					return nil, err
				}
			} else {
				if sale.GetSaleState() != pbd.SaleStatus_SOLD {
					sale.SaleState = pbd.SaleStatus_SOLD
					sale.SoldDate = soldDate
					sale.TimeRefreshed = now

					if item.GetCondition() != "" && sale.GetCondition() == "" {
						sale.Condition = item.GetCondition()
					}
					if item.GetSleeveCondition() != "" && sale.GetSleeveCondition() == "" {
						sale.SleeveCondition = item.GetSleeveCondition()
					}

					if item.GetPrice() != nil && (sale.GetCurrentPrice() == nil || sale.GetCurrentPrice().GetValue() != item.GetPrice().GetValue()) {
						sale.Updates = append(sale.Updates, &pb.PriceUpdate{
							Date:     now,
							SetPrice: item.GetPrice(),
						})
						sale.CurrentPrice = item.GetPrice()
						sale.LastPriceUpdate = now
					}
					tidyUpdates(sale)

					err = b.db.SaveSale(ctx, d.GetUserId(), sale)
					if err != nil {
						return nil, err
					}
					salesMarkedSold.Inc()
				}
			}
		}
	}

	return pagination, nil
}

func (b *BackgroundRunner) ProcessSyncOrders(ctx context.Context, d discogs.Discogs, user *pb.StoredUser, entry *pb.QueueElement, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	page := entry.GetSyncOrders().GetPage()
	if page == 0 {
		page = 1
	}
	syncId := entry.GetSyncOrders().GetSyncId()
	if syncId == 0 {
		syncId = time.Now().UnixNano()
		if entry.GetSyncOrders() != nil {
			entry.GetSyncOrders().SyncId = syncId
		}
	}

	pagination, err := b.SyncOrders(ctx, d, user, page)
	if err != nil {
		orderSyncTotal.With(prometheus.Labels{"status": "error"}).Inc()
		orderSyncErrors.With(prometheus.Labels{"code": fmt.Sprintf("%v", status.Code(err))}).Inc()
		return err
	}
	orderSyncTotal.With(prometheus.Labels{"status": "success"}).Inc()

	if pagination != nil && page < pagination.GetPages() {
		err = EnqueueWithIgnore(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
			Intention: entry.GetIntention(),
			RunDate:   time.Now().UnixNano() + 1,
			Force:     entry.GetForce(),
			Entry: &pb.QueueElement_SyncOrders{
				SyncOrders: &pb.SyncOrders{
					Page:   page + 1,
					SyncId: syncId,
				},
			},
			Auth: entry.GetAuth(),
		}}, enqueue)
		if err != nil {
			return fmt.Errorf("unable to enqueue next sync orders page: %w", err)
		}
	}

	if pagination == nil || page >= pagination.GetPages() {
		user.LastOrderSync = time.Now().UnixNano()
		err = b.db.SaveUser(ctx, user)
		if err != nil {
			return fmt.Errorf("unable to save user: %w", err)
		}

		err = EnqueueWithIgnore(ctx, &pb.EnqueueRequest{Element: &pb.QueueElement{
			Intention: entry.GetIntention(),
			RunDate:   time.Now().UnixNano() + 10,
			Entry: &pb.QueueElement_LinkSales{
				LinkSales: &pb.LinkSales{
					RefreshId: syncId,
				},
			},
			Auth: entry.GetAuth(),
		}}, enqueue)
		if err != nil {
			return fmt.Errorf("unable to enqueue link sales job: %w", err)
		}
	}

	return nil
}
