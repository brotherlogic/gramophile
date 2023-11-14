package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

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
			err := b.db.SaveSale(ctx, d.GetUserId(), &pb.SaleInfo{
				SaleId:          sale.GetSaleId(),
				LastPriceUpdate: time.Now().Unix(),
				SaleState:       sale.GetStatus(),
				ReleaseId:       sale.GetReleaseId(),
				Condition:       sale.GetCondition(),
				CurrentPrice: &pbd.Price{
					Value:    sale.GetPrice().GetValue(),
					Currency: sale.GetPrice().GetCurrency(),
				},
			})
			if err != nil {
				return nil, err
			}
		} else if status.Code(err) == codes.OK {
			csale.SaleState = sale.GetStatus()
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
	log.Printf("Adjusting with config: %v", c)
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
			if time.Since(time.Unix(sale.GetLastPriceUpdate(), 0)) > getUpdateTime(c) {
				nsp, err := adjustPrice(ctx, sale, c)
				if err != nil {
					return fmt.Errorf("unable to adjust price: %w", err)
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
		return fmt.Errorf("unable to update sale price: %w", err)
	}

	if sale.GetCurrentPrice() == nil {
		sale.CurrentPrice = &pbd.Price{Value: newprice}
	} else {
		sale.GetCurrentPrice().Value = newprice
	}
	sale.Updates = append(sale.Updates, &pb.PriceUpdate{Date: time.Now().Unix(), SetPrice: sale.GetCurrentPrice()})
	sale.LastPriceUpdate = time.Now().Unix()
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
