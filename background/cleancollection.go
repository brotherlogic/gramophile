package background

import (
	"context"
	"log"

	"github.com/brotherlogic/discogs"
)

func (b *BackgroundRunner) CleanCollection(ctx context.Context, d discogs.Discogs, refreshId int64) error {
	records, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		return err
	}

	for _, r := range records {
		record, err := b.db.GetRecord(ctx, d.GetUserId(), r)
		if err != nil {
			return err
		}

		if record.GetRefreshId() != refreshId {
			err = b.db.DeleteRecord(ctx, d.GetUserId(), r)
			if err != nil {
				return err
			}
		}
	}

	// Reset the refresh lock
	b.ReleaseRefresh = 0

	return nil
}

func (b *BackgroundRunner) CleanSales(ctx context.Context, userid int32, refreshId int64) error {
	log.Printf("Cleaning Sales for %v", userid)
	saleids, err := b.db.GetSales(ctx, userid)
	if err != nil {
		return err
	}

	for _, r := range saleids {
		sale, err := b.db.GetSale(ctx, userid, r)
		if err != nil {
			return err
		}

		if sale.GetRefreshId() != refreshId {
			log.Printf("Deleting %v since %v does not equal %v", sale.GetSaleId(), sale.GetRefreshId(), refreshId)
			b.db.DeleteSale(ctx, userid, r)
		}
	}

	return nil
}
