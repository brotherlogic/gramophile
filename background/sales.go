package background

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func tidyUpdates(s *pb.SaleInfo) {
	updates := s.GetUpdates()
	sort.SliceStable(updates, func(i, j int) bool {
		return updates[i].GetDate() < updates[j].GetDate()
	})

	var nupdates []*pb.PriceUpdate
	currPrice := int32(-100)

	for _, update := range updates {
		if currPrice != (update.GetSetPrice().GetValue()) {
			nupdates = append(nupdates, update)
			currPrice = update.GetSetPrice().GetValue()
		}
	}
	s.Updates = nupdates
}

func (b *BackgroundRunner) SyncSales(ctx context.Context, d discogs.Discogs, page int32, id int64) (*pbd.Pagination, error) {
	sales, pagination, err := d.ListSales(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("unable to list sales: %w", err)
	}

	if len(sales) > 0 {
		log.Printf("found %v sales -> %v", len(sales), sales[0])
	} else {
		log.Printf("found no sales :-(")
	}
	for _, sale := range sales {
		if sale.GetStatus() == pbd.SaleStatus_FOR_SALE {
			log.Printf("SALEITEM: %v", sale)
		} else {
			log.Printf("WHATSALEITEM: %v", sale)
		}

		csale, err := b.db.GetSale(ctx, d.GetUserId(), sale.GetSaleId())
		if status.Code(err) == codes.NotFound {
			log.Printf("Creating sale: %v, %v -> %v", d.GetUserId(), sale.GetSaleId(), err)
			err := b.db.SaveSale(ctx, d.GetUserId(), &pb.SaleInfo{
				SaleId:          sale.GetSaleId(),
				LastPriceUpdate: time.Now().UnixNano(),
				SaleState:       sale.GetStatus(),
				ReleaseId:       sale.GetReleaseId(),
				Condition:       sale.GetCondition(),
				CurrentPrice: &pbd.Price{
					Value:    sale.GetPrice().GetValue(),
					Currency: sale.GetPrice().GetCurrency(),
				},
				TimeCreated:   time.Now().UnixNano(),
				RefreshId:     id,
				TimeRefreshed: time.Now().UnixNano(),
			})
			if err != nil {
				return nil, err
			}
		} else if status.Code(err) == codes.OK {
			log.Printf("Updating sale: %v, %v -> %v (%v)", d.GetUserId(), sale.GetSaleId(), err, id)

			before := len(csale.GetUpdates())
			csale.SaleState = sale.GetStatus()
			csale.RefreshId = id
			csale.TimeRefreshed = time.Now().UnixNano()
			csale.Updates = append(csale.Updates, &pb.PriceUpdate{
				Date:     time.Now().UnixNano(),
				SetPrice: sale.GetPrice(),
			})
			csale.CurrentPrice = &pbd.Price{
				Value:    sale.GetPrice().GetValue(),
				Currency: sale.GetPrice().GetCurrency(),
			}

			tidyUpdates(csale)
			log.Printf("Setting sale state for %v and tidying updates: %v -> %v", csale.GetSaleId(), before, len(csale.GetUpdates()))
			err := b.db.SaveSale(ctx, d.GetUserId(), csale)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	return pagination, nil
}

func getUpdateTime(c *pb.SaleConfig) time.Duration {
	return time.Second * time.Duration(c.GetUpdateFrequencySeconds())
}

func max(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

func adjustPrice(ctx context.Context, s *pb.SaleInfo, c *pb.SaleConfig) (int32, error) {
	log.Printf("Adjusting %v with config: %v", s.GetSaleId(), c)
	switch c.GetUpdateType() {
	case pb.SaleUpdateType_MINIMAL_REDUCE:
		return s.GetCurrentPrice().Value - 1, nil
	case pb.SaleUpdateType_NO_SALE_UPDATE:
		return s.GetCurrentPrice().GetValue(), nil
	case pb.SaleUpdateType_REDUCE_TO_MEDIAN:
		// Bail if there is not current median price
		if s.GetMedianPrice().GetValue() == 0 {
			return s.GetCurrentPrice().GetValue(), nil
		}

		// Are we in post reduction time?
		if s.GetTimeAtMedian() > 0 && c.GetPostMedianReduction() > 0 {
			log.Printf("For %v in post reduction", s.GetSaleId())
			if time.Since(time.Unix(0, s.GetTimeAtMedian())).Seconds() > float64(c.GetPostMedianTime()) {
				postMedianCycles := int32(math.Floor((time.Since(time.Unix(0, s.GetTimeAtMedian())).Seconds() - float64(c.GetPostMedianTime())) / float64(c.GetPostMedianReductionFrequency())))
				log.Printf("Adjusting down from median: %v (%v / %v)", postMedianCycles, time.Since(time.Unix(0, s.GetTimeAtMedian())).Seconds()-float64(c.GetPostMedianTime()), c.GetPostMedianReductionFrequency())

				lowerBound := c.GetLowerBound()
				if c.GetLowerBoundStrategy() == pb.LowerBoundStrategy_DISCOGS_LOW {
					lowerBound = s.GetLowPrice().GetValue()
				}
				log.Printf("Found lower bound %v", lowerBound)
				if lowerBound > 0 {
					return max(s.GetMedianPrice().GetValue()-postMedianCycles*c.GetPostMedianReduction(), lowerBound), nil
				}
			} else {
				log.Printf("No time to adjust: (%v) %v vs %v", s.GetSaleId(), time.Since(time.Unix(0, s.GetTimeAtMedian())).Seconds(), c.GetPostMedianTime())
			}

		}

		return max(s.CurrentPrice.GetValue()-c.GetReduction(), s.GetMedianPrice().GetValue()), nil
	default:
		return 0, fmt.Errorf("unable to adjust price for %v", c.GetUpdateType())
	}
}

func (b *BackgroundRunner) AdjustSales(ctx context.Context, c *pb.SaleConfig, user *pb.StoredUser, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	sales, err := b.db.GetSales(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unable to get all sales: %w", err)
	}
	log.Printf("Adjusting %v sales", len(sales))

	for _, sid := range sales {
		sale, err := b.db.GetSale(ctx, user.GetUser().GetDiscogsUserId(), sid)
		if err != nil {
			return fmt.Errorf("unable to read sale: %w", err)
		}

		if sale.GetSaleState() == pbd.SaleStatus_FOR_SALE {
			if time.Since(time.Unix(0, sale.GetLastPriceUpdate())) > getUpdateTime(c) {
				log.Printf("Working off of: %v", sale)
				nsp, err := adjustPrice(ctx, sale, c)
				if err != nil {
					return fmt.Errorf("unable to adjust price: %w", err)
				}

				// If we've reached the median price, then explicitly set this
				if c.GetUpdateType() == pb.SaleUpdateType_REDUCE_TO_MEDIAN &&
					nsp == sale.GetMedianPrice().GetValue() && sale.TimeAtMedian == 0 {
					sale.TimeAtMedian = time.Now().UnixNano()

					err = b.db.SaveSale(ctx, user.GetUser().GetDiscogsUserId(), sale)
					if err != nil {
						return err
					}
				}

				log.Printf("ADJUST PRICE(%v) %v -> %v", sale.GetSaleId(), sale.GetCurrentPrice().GetValue(), nsp)

				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate: time.Now().UnixNano(),
						Auth:    user.GetAuth().GetToken(),
						Entry: &pb.QueueElement_UpdateSale{
							UpdateSale: &pb.UpdateSale{
								SaleId:    sid,
								NewPrice:  nsp,
								ReleaseId: sale.GetReleaseId(),
								Condition: sale.GetCondition(),
							}}},
				})
				if err != nil {
					return fmt.Errorf("unable to queue sales: %v", err)
				}
			} else {
				log.Printf("Not adjusting %v since %v is less than %v", sale.GetSaleId(), time.Since(time.Unix(0, sale.GetLastPriceUpdate())), getUpdateTime(c))
			}
		} else {
			log.Printf("%v is not for sale", sid)
		}
	}

	return nil
}

func (b *BackgroundRunner) UpdateSalePrice(ctx context.Context, d discogs.Discogs, sid int64, releaseid int64, condition string, newprice int32) error {
	sale, err := b.db.GetSale(ctx, d.GetUserId(), sid)
	if err != nil {
		return fmt.Errorf("unable to load sale: %w", err)
	}

	log.Printf("Updating price %v -> %v", sale, newprice)
	err = d.UpdateSale(ctx, sid, releaseid, condition, newprice)
	if err != nil {
		// We expect FailedPrecondition if the sale status has chnaged since the sale price request went in (e.g. the
		// item has sold or take off the marketplace). In this case we silently succeed
		if status.Code(err) == codes.FailedPrecondition {
			return nil
		}
		return fmt.Errorf("unable to update sale price: %w", err)
	}

	if sale.GetCurrentPrice() == nil {
		sale.CurrentPrice = &pbd.Price{Value: newprice}
	} else {
		sale.GetCurrentPrice().Value = newprice
	}
	sale.Updates = append(sale.Updates, &pb.PriceUpdate{Date: time.Now().UnixNano(), SetPrice: sale.GetCurrentPrice()})
	sale.LastPriceUpdate = time.Now().UnixNano()
	return b.db.SaveSale(ctx, d.GetUserId(), sale)
}

func (b *BackgroundRunner) LinkSales(ctx context.Context, user *pb.StoredUser) error {
	iids, err := b.db.GetRecords(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unable to read records: %w", err)
	}

	var records []*pb.Record
	for _, r := range iids {
		rec, err := b.db.GetRecord(ctx, user.GetUser().GetDiscogsUserId(), r)
		if err != nil {
			return fmt.Errorf("error on record read: %w", err)
		}

		records = append(records, rec)
	}

	sids, err := b.db.GetSales(ctx, user.GetUser().GetDiscogsUserId())
	if err != nil {
		return fmt.Errorf("unable to read sales: %w", err)
	}
	var sales []*pb.SaleInfo
	for _, s := range sids {
		sale, err := b.db.GetSale(ctx, user.GetUser().GetDiscogsUserId(), s)
		log.Printf("LOADED: %v", sale)
		if err != nil {
			return fmt.Errorf("unable to read sale: %w", err)
		}

		log.Printf("Got Sale: %v", sale)
		sales = append(sales, sale)
	}

	return b.HardLink(ctx, user, records, sales)
}

func (b *BackgroundRunner) HardLink(ctx context.Context, user *pb.StoredUser, records []*pb.Record, sales []*pb.SaleInfo) error {
	for _, sale := range sales {
		for _, record := range records {
			changed := false
			sale_changed := false
			if record.GetRelease().GetId() == sale.GetReleaseId() {
				log.Printf("LINK %v or %v ($%v)", record.GetRelease().GetInstanceId(), record.GetSaleId(), sale)

				// Ensure we copy over any changes to the median price
				if record.GetMedianPrice().GetValue() != sale.GetMedianPrice().GetValue() {
					sale.MedianPrice = record.GetMedianPrice()
					sale_changed = true
				}

				if record.GetLowPrice().GetValue() != sale.GetLowPrice().GetValue() {
					sale.LowPrice = record.GetLowPrice()
					sale_changed = true
				}

				if record.GetSaleId() != sale.GetSaleId() {
					record.SaleId = sale.GetSaleId()

					changed = true
				}
			}

			if changed {
				log.Printf("Saving on change: %v", record)
				err := b.db.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), record)
				if err != nil {
					return fmt.Errorf("unable to save record: %w", err)
				}
			}
			if sale_changed {
				log.Printf("Saving sale on change: %v", record)
				err := b.db.SaveSale(ctx, user.GetUser().GetDiscogsUserId(), sale)
				if err != nil {
					return fmt.Errorf("unable to save sale info: %w", err)
				}
			}
		}
	}

	return nil
}
