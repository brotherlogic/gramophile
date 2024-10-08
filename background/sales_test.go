package background

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	rstore_client "github.com/brotherlogic/rstore/client"
)

func TestUpdate(t *testing.T) {
	updatesb := []*pb.PriceUpdate{
		{
			Date:     1,
			SetPrice: &pbd.Price{Value: 100},
		},
		{
			Date:     2,
			SetPrice: &pbd.Price{Value: 200},
		},
		{
			Date:     3,
			SetPrice: &pbd.Price{Value: 200},
		},
		{
			Date:     4,
			SetPrice: &pbd.Price{Value: 300},
		},
	}

	saleInfo := &pb.SaleInfo{Updates: updatesb}
	tidyUpdates(saleInfo)

	if len(saleInfo.GetUpdates()) != 3 {
		t.Errorf("Should have just 3 updates: %v", len(saleInfo.GetUpdates()))
	}

	updates := saleInfo.GetUpdates()
	sort.SliceStable(updates, func(i, j int) bool {
		return updates[i].GetDate() < updates[j].GetDate()
	})

	if updates[0].Date != 1 || updates[0].SetPrice.Value != 100 {
		t.Errorf("Bad update: %v", updates[0])
	}
	if updates[1].Date != 2 || updates[1].SetPrice.Value != 200 {
		t.Errorf("Bad update: %v", updates[1])
	}
	if updates[2].Date != 4 || updates[2].SetPrice.Value != 300 {
		t.Errorf("Bad update: %v", updates[2])
	}
}

func TestFirstUpdate(t *testing.T) {
	updatesb := []*pb.PriceUpdate{
		{
			Date:     1,
			SetPrice: &pbd.Price{Value: 100},
		},
		{
			Date:     2,
			SetPrice: &pbd.Price{Value: 100},
		},
	}

	saleInfo := &pb.SaleInfo{Updates: updatesb}

	tidyUpdates(saleInfo)

	if len(saleInfo.GetUpdates()) != 1 {
		t.Errorf("Should have just 1 update: %v", len(saleInfo.GetUpdates()))
	}

	updates := saleInfo.GetUpdates()
	if updates[0].Date != 1 || updates[0].SetPrice.Value != 100 {
		t.Errorf("Bad update: %v", updates[0])
	}
}

var reductionTests = []struct {
	name           string
	startPrice     int32
	medianPrice    int32
	lowPrice       int32
	expectedPrice  int32
	timeSinceStart int32
	timeToMedian   int32
	timeToLow      int32
}{
	{
		name:           "Half Way Median",
		startPrice:     100,
		medianPrice:    50,
		lowPrice:       10,
		expectedPrice:  75,
		timeSinceStart: 5,
		timeToMedian:   10,
		timeToLow:      10,
	},
}

func TestReduction(t *testing.T) {
	for _, test := range reductionTests {
		config := &pb.SaleConfig{
			TimeToMedianDays: test.timeToMedian,
			TimeToLowerDays:  test.timeToLow,
			UpdateType:       pb.SaleUpdateType_REDUCE_TO_MEDIAN,
		}
		sale := &pb.SaleInfo{
			InitialPrice: &pbd.Price{Value: test.startPrice},
			ListedDate:   time.Now().Add(-time.Hour * time.Duration(24*test.timeSinceStart)).UnixNano(),
			MedianPrice:  &pbd.Price{Value: test.medianPrice},
			LowPrice:     &pbd.Price{Value: test.lowPrice},
		}

		nPrice, _, err := adjustPrice(context.Background(), sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN)
		if err != nil {
			t.Errorf("Bad price reduction: %v", err)
		}
		if nPrice != test.expectedPrice {
			t.Errorf("Price was %v, expected %v (%v)", nPrice, test.expectedPrice, test.name)
		}
	}
}

func TestSkipOnSame(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)
	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}}}}

	d.SaveSale(context.Background(), 1234, &pb.SaleInfo{ReleaseId: 123, SaleId: 12345, CurrentPrice: &pbd.Price{Value: 123}})

	b := GetBackgroundRunner(d, "", "", "")
	err := b.UpdateSalePrice(context.Background(), di, 12345, 123, "Very Good Plus (VG+)", 123, "Testing")
	if err == nil {
		t.Errorf("Should have failed: %v", err)
	}
}

func TestSold(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}, Status: pbd.SaleStatus_SOLD}}}

	d.SaveSale(context.Background(), 123, &pb.SaleInfo{ReleaseId: 123, SaleId: 12345, CurrentPrice: &pbd.Price{Value: 123}, SaleState: pbd.SaleStatus_FOR_SALE})

	b := GetBackgroundRunner(d, "", "", "")

	_, err := b.SyncSales(context.Background(), di, 1, time.Now().UnixNano())
	if err != nil {
		t.Fatalf("Bad sync: %v", err)
	}

	sale, err := d.GetSale(context.Background(), 123, 12345)
	if err != nil {
		t.Fatalf("Unable to get sale: %v", err)
	}

	if sale.GetSoldDate() == 0 {
		t.Errorf("Sold date was not changed: %v", sale)
	}
}

func TestTypeOverride_NoOverride(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)

	recordedPrice := int32(0)

	/*di := &discogs.TestDiscogsClient{
	UserId: 123,
	Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
	Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}, Status: pbd.SaleStatus_SOLD}}}
	*/
	d.SaveSale(context.Background(), 123,
		&pb.SaleInfo{
			ReleaseId:    123,
			SaleId:       12345,
			TimeAtMedian: time.Now().Add(-time.Hour * 48).UnixNano(),
			CurrentPrice: &pbd.Price{Value: 200},
			MedianPrice:  &pbd.Price{Value: 200},
			LowPrice:     &pbd.Price{Value: 50},
			SaleState:    pbd.SaleStatus_FOR_SALE})

	b := GetBackgroundRunner(d, "", "", "")

	b.AdjustSales(context.Background(), &pb.SaleConfig{
		PostLowTime:                  1,
		PostMedianReduction:          50,
		LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
		PostMedianReductionFrequency: 10,
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
	}, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		//pass
		recordedPrice = req.GetElement().GetUpdateSale().GetNewPrice()
		return &pb.EnqueueResponse{}, nil
	})

	sale, err := d.GetSale(context.Background(), 123, 12345)
	if err != nil {
		t.Fatalf("Bad sale load: %v", err)
	}
	if recordedPrice != 200 {
		t.Errorf("Price was  adjusted: %v (%v)", sale, recordedPrice)
	}

}
func TestTypeOverride_Override(t *testing.T) {
	rstore := rstore_client.GetTestClient()
	d := db.NewTestDB(rstore)

	recordedPrice := int32(0)

	/*di := &discogs.TestDiscogsClient{
	UserId: 123,
	Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
	Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}, Status: pbd.SaleStatus_SOLD}}}
	*/
	d.SaveSale(context.Background(), 123,
		&pb.SaleInfo{
			ReleaseId:          123,
			SaleId:             12345,
			TimeAtMedian:       time.Now().Add(-time.Hour * 48).UnixNano(),
			CurrentPrice:       &pbd.Price{Value: 200},
			MedianPrice:        &pbd.Price{Value: 200},
			LowPrice:           &pbd.Price{Value: 50},
			SaleUpdateOverride: pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
			SaleState:          pbd.SaleStatus_FOR_SALE})

	b := GetBackgroundRunner(d, "", "", "")

	b.AdjustSales(context.Background(), &pb.SaleConfig{
		PostLowTime:                  1,
		PostMedianReduction:          50,
		LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
		PostMedianReductionFrequency: 10,
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN,
	}, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		//pass
		recordedPrice = req.GetElement().GetUpdateSale().GetNewPrice()
		return &pb.EnqueueResponse{}, nil
	})

	sale, err := d.GetSale(context.Background(), 123, 12345)
	if err != nil {
		t.Fatalf("Bad sale load: %v", err)
	}
	if recordedPrice != 50 {
		t.Errorf("Price was not adjusted: %v", sale)
	}

}
