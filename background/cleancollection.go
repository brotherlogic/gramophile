package background

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/brotherlogic/discogs"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (b *BackgroundRunner) DeleteRecord(ctx context.Context, d discogs.Discogs, iid int64) error {
	record, err := b.db.GetRecord(ctx, d.GetUserId(), iid)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return nil
		}
		return err
	}

	releases, p, err := d.GetCollectionRelease(ctx, record.GetRelease().GetId(), 1)
	if err != nil {
		return err
	}

	found := false
	for _, rel := range releases {
		if rel.GetInstanceId() == iid {
			found = true
			break
		}
	}

	for page := int32(2); page <= p.GetPages(); page++ {
		if found {
			break
		}
		releases, _, err = d.GetCollectionRelease(ctx, record.GetRelease().GetId(), page)
		if err != nil {
			return err
		}
		for _, rel := range releases {
			if rel.GetInstanceId() == iid {
				found = true
				break
			}
		}
	}

	if !found {
		return b.db.DeleteRecord(ctx, d.GetUserId(), iid)
	}

	return nil
}

func (b *BackgroundRunner) CleanCollection(ctx context.Context, d discogs.Discogs, refreshId int64, authToken string, enqueue func(context.Context, *pb.EnqueueRequest) (*pb.EnqueueResponse, error)) error {
	qlog(ctx, "Cleaning collection with %v - > %v", refreshId, d.GetUserId())
	records, err := b.db.GetRecords(ctx, d.GetUserId())
	if err != nil {
		qlog(ctx, "Got records: %v", err)
		return err
	}

	qlog(ctx, "Cleaning %v records", len(records))
	for _, r := range records {
		record, err := b.db.GetRecord(ctx, d.GetUserId(), r)
		if err != nil {
			qlog(ctx, "Failed on get record %v", err)
			return err
		}

		if record.GetRefreshId() < refreshId {
			log.Printf("ENQUEUE DELETE %v because %v != %v", r, record.GetRefreshId(), refreshId)
			_, err = enqueue(ctx, &pb.EnqueueRequest{
				Element: &pb.QueueElement{
					RunDate:   time.Now().UnixNano(),
					Auth:      authToken,
					Intention: fmt.Sprintf("Delete record %v", r),
					Entry: &pb.QueueElement_DeleteRecord{
						DeleteRecord: &pb.DeleteRecord{
							Iid: r,
						}},
				},
			})
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
