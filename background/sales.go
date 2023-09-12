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

	log.Printf("found %v sales", len(sales))
	for _, sale := range sales {
		b.db.SaveSale(ctx, d.GetUserId(), &pb.SaleInfo{
			SaleId:          sale.GetSaleId(),
			LastPriceUpdate: time.Now().Unix(),
			SaleState:       sale.GetStatus(),
			ReleaseId:       sale.GetReleaseId(),
		})
	}

	return pagination, nil
}

func getUpdateTime(c *pb.SaleConfig) time.Duration {
	return time.Second * time.Duration(c.GetUpdateFrequencySeconds())
}

func adjustPrice(ctx context.Context, s *pb.SaleInfo, c *pb.SaleConfig) (int32, error) {
	switch c.GetUpdateType() {
	case pb.SaleUpdateType_MINIMAL_REDUCE:
		return s.GetCurrentPrice().Value - 1, nil
	case pb.SaleUpdateType_SALE_UPDATE_UNKNOWN:
		return s.GetCurrentPrice().GetValue(), nil
	default:
		return 0, fmt.Errorf("unable to adjust price for %v", c.GetUpdateType())
	}
}

func (b *BackgroundRunner) AdjustSales(ctx context.Context, c *pb.SaleConfig, di discogs.Discogs) error {
	sales, err := b.db.GetSales(ctx, di.GetUserId())
	if err != nil {
		return fmt.Errorf("unable to get all sales: %w", err)
	}

	for _, sid := range sales {
		sale, err := b.db.GetSale(ctx, di.GetUserId(), sid)
		if err != nil {
			return fmt.Errorf("unable to read sale: %w", err)
		}

		if sale.GetSaleState() == pbd.SaleStatus_FOR_SALE {
			if time.Since(time.Unix(sale.GetLastPriceUpdate(), 0)) > getUpdateTime(c) {
				nsp, err := adjustPrice(ctx, sale, c)
				if err != nil {
					return fmt.Errorf("unable to adjust price: %w", err)
				}

				sale.NewPrice = nsp
				b.db.SaveSale(ctx, di.GetUserId(), sale)
			}
		}
	}

	return nil
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
		if err != nil {
			return fmt.Errorf("unable to read sale: %w", err)
		}

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
				log.Printf("%v or %v", record.GetSaleInfo(), record.GetSaleInfo().GetSaleId())
				if record.GetSaleInfo() != nil && record.GetSaleInfo().GetSaleId() == sale.GetSaleId() {
					if record.GetSaleInfo().GetSaleState() != sale.GetSaleState() {
						record.GetSaleInfo().SaleState = sale.GetSaleState()
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
