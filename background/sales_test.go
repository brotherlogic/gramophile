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

		nPrice, _, err := adjustPrice(context.Background(), sale, config)
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
