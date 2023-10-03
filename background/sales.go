package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func (b *BackgroundRunner) SyncSales(ctx context.Context, d discogs.Discogs, page int32, id int64) (*pbd.Pagination, error) {
	sales, pagination, err := d.ListSales(ctx, page)
	if err != nil {
		return nil, fmt.Errorf("unable to list sales: %w", err)
	}

	log.Printf("found %v sales -> %v", len(sales), sales[0])
	for _, sale := range sales {
		b.db.SaveSale(ctx, d.GetUserId(), &pb.SaleInfo{
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
	}

	return pagination, nil
}

func getUpdateTime(c *pb.SaleConfig) time.Duration {
	return time.Second * time.Duration(c.GetUpdateFrequencySeconds())
}

func adjustPrice(ctx context.Context, s *pb.SaleInfo, c *pb.SaleConfig) (int32, error) {
	log.Printf("Adjusting with config: %v", c)
	switch c.GetUpdateType() {
	case pb.SaleUpdateType_MINIMAL_REDUCE:
		return s.GetCurrentPrice().Value - 1, nil
	case pb.SaleUpdateType_SALE_UPDATE_UNKNOWN:
		return s.GetCurrentPrice().GetValue(), nil
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

				log.Printf("ADJUST PRICE %v -> %v", sale.GetCurrentPrice().GetValue(), nsp)

				_, err = enqueue(ctx, &pb.EnqueueRequest{
					Element: &pb.QueueElement{
						RunDate: time.Now().Unix(),
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
	log.Printf("Hard Link with %v records and %v sales", (records), (sales))

	for _, sale := range sales {
		for _, record := range records {
			changed := false
			if record.GetRelease().GetId() == sale.GetReleaseId() {
				log.Printf("%v or %v ($%v)", record.GetSaleInfo(), record.GetSaleInfo().GetSaleId(), sale)
				if record.GetSaleInfo() != nil && record.GetSaleInfo().GetCurrentPrice() != nil && record.GetSaleInfo().GetSaleId() == sale.GetSaleId() {
					if record.GetSaleInfo().GetSaleState() != sale.GetSaleState() {
						record.GetSaleInfo().SaleState = sale.GetSaleState()
						changed = true
					}
					if record.GetSaleInfo().GetCurrentPrice().GetValue() != sale.GetCurrentPrice().GetValue() {
						record.GetSaleInfo().GetCurrentPrice().Value = sale.GetCurrentPrice().GetValue()
						changed = true
					}
				} else {
					record.SaleInfo = sale
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
		}
	}

	return nil
}
