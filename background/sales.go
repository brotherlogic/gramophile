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
				b.db.SaveRecord(ctx, user.GetUser().GetDiscogsUserId(), record)
			}
		}
	}

	return nil
}
