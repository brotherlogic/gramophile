package background

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/brotherlogic/discogs"
	pbd "github.com/brotherlogic/discogs/proto"
	ghbclient "github.com/brotherlogic/githubridge/client"
	ghbpb "github.com/brotherlogic/githubridge/proto"
	"github.com/brotherlogic/gramophile/db"
	pb "github.com/brotherlogic/gramophile/proto"
	pstore_client "github.com/brotherlogic/pstore/client"
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
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
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
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Fields: []*pbd.Field{{Id: 10, Name: "Keep"}},
		Sales:  []*pbd.SaleItem{{ReleaseId: 123, SaleId: 12345, Price: &pbd.Price{Value: 1234, Currency: "USD"}, Status: pbd.SaleStatus_SOLD}}}

	d.SaveSale(context.Background(), 123, &pb.SaleInfo{ReleaseId: 123, SaleId: 12345, CurrentPrice: &pbd.Price{Value: 123}, SaleState: pbd.SaleStatus_FOR_SALE})

	b := GetBackgroundRunner(d, "", "", "")

	_, _, err := b.SyncSales(context.Background(), di, 1, time.Now().UnixNano(), 0)
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
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

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
	}, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
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
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

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
	}, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
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

func TestHardLink_PicksLatestSale(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	db := db.NewTestDB(pstore)
	b := GetBackgroundRunner(db, "", "", "")

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}
	records := []*pb.Record{
		{
			Release: &pbd.Release{Id: 100, InstanceId: 1000},
			SaleId:  0,
		},
	}
	sales := []*pb.SaleInfo{
		{
			SaleId:    50,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_FOR_SALE,
		},
		{
			SaleId:    100,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_FOR_SALE,
		},
		{
			SaleId:    75,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_FOR_SALE,
		},
	}

	ctx := context.Background()
	err := b.HardLink(ctx, user, records, sales)
	if err != nil {
		t.Fatalf("HardLink failed: %v", err)
	}

	if records[0].SaleId != 100 {
		t.Errorf("Expected SaleId 100, got %v", records[0].SaleId)
	}
}

func TestHardLink_OnlyValidStates(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	db := db.NewTestDB(pstore)
	b := GetBackgroundRunner(db, "", "", "")

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}
	records := []*pb.Record{
		{
			Release: &pbd.Release{Id: 100, InstanceId: 1000},
			SaleId:  50, // Currently linked to an old sale
		},
		{
			Release: &pbd.Release{Id: 200, InstanceId: 2000},
			SaleId:  200, // Linked to a sale that will become invalid
		},
	}
	sales := []*pb.SaleInfo{
		{
			SaleId:    50,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_SOLD, // Valid
		},
		{
			SaleId:    75,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_FOR_SALE, // Valid and latest
		},
		{
			SaleId:    80,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus(100), // Invalid, should be ignored
		},
		{
			SaleId:    200,
			ReleaseId: 200,
			SaleState: pbd.SaleStatus(100), // Invalid
		},
	}

	ctx := context.Background()
	err := b.HardLink(ctx, user, records, sales)
	if err != nil {
		t.Fatalf("HardLink failed: %v", err)
	}

	// Record 100 should be linked to 75 (latest valid)
	if records[0].SaleId != 75 {
		t.Errorf("Expected SaleId 75 for record 0, got %v", records[0].SaleId)
	}

	// Record 200 should have SaleId cleared (no valid sales)
	if records[1].SaleId != 0 {
		t.Errorf("Expected SaleId 0 for record 1, got %v", records[1].SaleId)
	}
}

func TestAdjustPrice_MissingMedian(t *testing.T) {
	ctx := context.Background()
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 500},
		MedianPrice:  &pbd.Price{Value: 0},
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType: pb.SaleUpdateType_REDUCE_TO_MEDIAN,
		Reduction:  50,
	}
	_, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN)
	if !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("expected ErrMissingMetadata, got: %v", err)
	}
}

func TestAdjustPrice_MissingLowPrice(t *testing.T) {
	ctx := context.Background()
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 500},
		MedianPrice:  &pbd.Price{Value: 400},
		LowPrice:     &pbd.Price{Value: 0},
		TimeAtMedian: time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
		PostMedianTime:               1,
		PostMedianReduction:          50,
		PostMedianReductionFrequency: 10,
		LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
	}
	_, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
	if !errors.Is(err, ErrMissingMetadata) {
		t.Fatalf("expected ErrMissingMetadata, got: %v", err)
	}
}

func TestAdjustPrice_PostMedian_ImmediateFirstCycleReduction(t *testing.T) {
	ctx := context.Background()
	// Elapsed time since median is postMedianTime (100s) + 1s = 101s.
	// Within the first frequency window (frequency = 10s), postMedianCycles should be 1.
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 400},
		MedianPrice:  &pbd.Price{Value: 400},
		TimeAtMedian: time.Now().Add(-101 * time.Second).UnixNano(),
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
		PostMedianTime:               100,
		PostMedianReduction:          50,
		PostMedianReductionFrequency: 10,
		LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
		LowerBound:                   100,
	}
	newPrice, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 400 - (1 * 50) = 350
	if newPrice != 350 {
		t.Errorf("expected newPrice 350 on immediate first cycle, got %v", newPrice)
	}
}

func TestAdjustPrice_PostMedian_MultipleCyclesReduction(t *testing.T) {
	ctx := context.Background()
	// Elapsed time since median is postMedianTime (100s) + 25s = 125s.
	// With frequency = 10s: math.Floor(25/10) + 1 = 3 cycles.
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 400},
		MedianPrice:  &pbd.Price{Value: 400},
		TimeAtMedian: time.Now().Add(-125 * time.Second).UnixNano(),
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
		PostMedianTime:               100,
		PostMedianReduction:          50,
		PostMedianReductionFrequency: 10,
		LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
		LowerBound:                   100,
	}
	newPrice, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Expected: 400 - (3 * 50) = 250
	if newPrice != 250 {
		t.Errorf("expected newPrice 250 on multiple cycles, got %v", newPrice)
	}
}

func TestAdjustPrice_PostMedian_RespectsLowerBound(t *testing.T) {
	ctx := context.Background()

	// Subtest 1: Config lower bound
	t.Run("ConfigLowerBound", func(t *testing.T) {
		sale := &pb.SaleInfo{
			SaleId:       1234,
			ReleaseId:    100,
			CurrentPrice: &pbd.Price{Value: 400},
			MedianPrice:  &pbd.Price{Value: 400},
			TimeAtMedian: time.Now().Add(-300 * time.Second).UnixNano(),
			SaleState:    pbd.SaleStatus_FOR_SALE,
		}
		config := &pb.SaleConfig{
			UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
			PostMedianTime:               100,
			PostMedianReduction:          50,
			PostMedianReductionFrequency: 10,
			LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
			LowerBound:                   250,
		}
		// 200s elapsed post-median -> 21 cycles -> 400 - 1050 = -650 -> clamped to 250
		newPrice, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newPrice != 250 {
			t.Errorf("expected clamped price 250, got %v", newPrice)
		}
	})

	// Subtest 2: Discogs Low lower bound
	t.Run("DiscogsLowLowerBound", func(t *testing.T) {
		sale := &pb.SaleInfo{
			SaleId:       1234,
			ReleaseId:    100,
			CurrentPrice: &pbd.Price{Value: 400},
			MedianPrice:  &pbd.Price{Value: 400},
			LowPrice:     &pbd.Price{Value: 220},
			TimeAtMedian: time.Now().Add(-300 * time.Second).UnixNano(),
			SaleState:    pbd.SaleStatus_FOR_SALE,
		}
		config := &pb.SaleConfig{
			UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
			PostMedianTime:               100,
			PostMedianReduction:          50,
			PostMedianReductionFrequency: 10,
			LowerBoundStrategy:           pb.LowerBoundStrategy_DISCOGS_LOW,
		}
		// 200s elapsed post-median -> 21 cycles -> 400 - 1050 = -650 -> clamped to LowPrice (220)
		newPrice, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if newPrice != 220 {
			t.Errorf("expected clamped price 220, got %v", newPrice)
		}
	})
}

func TestAdjustPrice_PostMedian_HoldingBeforePostMedianTime(t *testing.T) {
	ctx := context.Background()
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 400},
		MedianPrice:  &pbd.Price{Value: 400},
		TimeAtMedian: time.Now().Add(-50 * time.Second).UnixNano(),
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
		PostMedianTime:               100,
		PostMedianReduction:          50,
		PostMedianReductionFrequency: 10,
		LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
		LowerBound:                   100,
	}
	// Has not reached postMedianTime (50s < 100s). Price should remain at median.
	newPrice, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if newPrice != 400 {
		t.Errorf("expected price to hold at 400, got %v", newPrice)
	}
}

func TestAdjustPrice_PostMedian_ZeroFrequencyGuard(t *testing.T) {
	ctx := context.Background()
	sale := &pb.SaleInfo{
		SaleId:       1234,
		ReleaseId:    100,
		CurrentPrice: &pbd.Price{Value: 400},
		MedianPrice:  &pbd.Price{Value: 400},
		TimeAtMedian: time.Now().Add(-150 * time.Second).UnixNano(),
		SaleState:    pbd.SaleStatus_FOR_SALE,
	}
	config := &pb.SaleConfig{
		UpdateType:                   pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
		PostMedianTime:               100,
		PostMedianReduction:          50,
		PostMedianReductionFrequency: 0,
		LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW,
		LowerBound:                   100,
	}
	_, _, err := adjustPrice(ctx, sale, config, pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW)
	if err == nil {
		t.Fatalf("expected error for zero frequency, got nil")
	}
}


func TestAdjustSales_SkipsMissingMetadataWithoutTimestampAdvance(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	initialLastPriceUpdate := time.Now().Add(-time.Hour * 72).UnixNano()
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          12345,
		ReleaseId:       100,
		CurrentPrice:    &pbd.Price{Value: 500},
		MedianPrice:     &pbd.Price{Value: 0}, // Missing metadata
		LastPriceUpdate: initialLastPriceUpdate,
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})

	enqueued := false
	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
	}
	config := &pb.SaleConfig{
		UpdateFrequencySeconds: 3600,
		UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
		Reduction:              50,
	}

	err := b.AdjustSales(ctx, config, user, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueued = true
		return &pb.EnqueueResponse{}, nil
	})
	if err != nil {
		t.Fatalf("AdjustSales failed: %v", err)
	}

	if enqueued {
		t.Errorf("expected no update task to be enqueued for sale missing metadata")
	}

	sale, err := d.GetSale(ctx, 123, 12345)
	if err != nil {
		t.Fatalf("unable to get sale: %v", err)
	}
	if sale.GetLastPriceUpdate() != initialLastPriceUpdate {
		t.Errorf("expected LastPriceUpdate to remain %v, got %v", initialLastPriceUpdate, sale.GetLastPriceUpdate())
	}

	savedUser, err := d.GetUser(ctx, user.GetAuth().GetToken())
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	if savedUser.GetLastSaleAdjust() == 0 {
		t.Errorf("expected user.LastSaleAdjust to be set > 0, got %v", savedUser.GetLastSaleAdjust())
	}
}

func TestProcessRefreshSales_Decoupled(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	di := &discogs.TestDiscogsClient{
		UserId: 123,
		Sales:  []*pbd.SaleItem{{ReleaseId: 100, SaleId: 12345, Price: &pbd.Price{Value: 1000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE}},
	}

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              50,
			},
		},
	}
	d.SaveUser(ctx, user)

	entry := &pb.QueueElement{
		Auth:  user.GetAuth().GetToken(),
		Force: true,
		Entry: &pb.QueueElement_RefreshSales{
			RefreshSales: &pb.RefreshSales{
				Page:      1,
				RefreshId: time.Now().UnixNano(),
			},
		},
	}

	var enqueuedIntents []string
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		if req.GetElement().GetUpdateSale() != nil {
			enqueuedIntents = append(enqueuedIntents, "UpdateSale")
		}
		return &pb.EnqueueResponse{}, nil
	}

	err := b.ProcessRefreshSales(ctx, di, user, entry, enqueue)
	if err != nil {
		t.Fatalf("ProcessRefreshSales failed: %v", err)
	}

	for _, intent := range enqueuedIntents {
		if intent == "UpdateSale" {
			t.Errorf("ProcessRefreshSales should not have enqueued AdjustSales / UpdateSale")
		}
	}
}

func TestAdjustSalesHandler(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_auth_token"},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				UpdateFrequencySeconds: 10,
				UpdateType:             pb.SaleUpdateType_REDUCE_TO_MEDIAN,
				Reduction:              50,
			},
		},
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("unable to save user: %v", err)
	}

	entry := &pb.QueueElement{
		Auth: "test_auth_token",
		Entry: &pb.QueueElement_AdjustSales{
			AdjustSales: &pb.AdjustSales{},
		},
	}

	handler, err := b.getHandler(entry)
	if err != nil {
		t.Fatalf("handler not registered for QueueElement_AdjustSales: %v", err)
	}

	dedupKey := handler.GetDeduplicationKey(entry)
	if dedupKey != "AdjustSales-test_auth_token" {
		t.Errorf("expected deduplication key 'AdjustSales-test_auth_token', got '%v'", dedupKey)
	}

	err = handler.Validate(ctx, d, entry)
	if err != nil {
		t.Errorf("expected Validate to return nil, got %v", err)
	}

	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return &pb.EnqueueResponse{}, nil
	}

	err = b.Execute(ctx, &discogs.TestDiscogsClient{}, user, entry, enqueue)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	savedUser, err := d.GetUser(ctx, user.GetAuth().GetToken())
	if err != nil {
		t.Fatalf("unable to get user: %v", err)
	}
	if savedUser.GetLastSaleAdjust() == 0 {
		t.Errorf("expected user.LastSaleAdjust to be set > 0, got %v", savedUser.GetLastSaleAdjust())
	}
}

type failingGetSalesDB struct {
	db.Database
}

func (f *failingGetSalesDB) GetSales(ctx context.Context, userId int32) ([]int64, error) {
	return nil, errors.New("simulated GetSales failure")
}

func TestAdjustSales_ResilientLoopContinuesOnIndividualError(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Sale 1: Will fail in adjustPrice due to unknown update type
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:             100,
		ReleaseId:          1000,
		CurrentPrice:       &pbd.Price{Value: 500},
		MedianPrice:        &pbd.Price{Value: 400},
		LastPriceUpdate:    time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:          pbd.SaleStatus_FOR_SALE,
		SaleUpdateOverride: pb.SaleUpdateType(999), // Invalid update type -> error in adjustPrice
	})

	// Sale 2: Valid sale, should be adjusted and enqueued
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:             200,
		ReleaseId:          2000,
		CurrentPrice:       &pbd.Price{Value: 500},
		MedianPrice:        &pbd.Price{Value: 400},
		LastPriceUpdate:    time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:          pbd.SaleStatus_FOR_SALE,
		SaleUpdateOverride: pb.SaleUpdateType_REDUCE_TO_MEDIAN,
	})

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
	}
	d.SaveUser(ctx, user)

	config := &pb.SaleConfig{
		UpdateFrequencySeconds: 10,
		Reduction:              50,
	}

	var enqueuedSales []int64
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		if req.GetElement().GetUpdateSale() != nil {
			enqueuedSales = append(enqueuedSales, req.GetElement().GetUpdateSale().GetSaleId())
		}
		return &pb.EnqueueResponse{}, nil
	}

	err := b.AdjustSales(ctx, config, user, enqueue)
	if err != nil {
		t.Fatalf("expected AdjustSales to return nil (resilient loop), got: %v", err)
	}

	// Verify Sale 200 was enqueued
	if len(enqueuedSales) != 1 || enqueuedSales[0] != 200 {
		t.Errorf("expected only Sale 200 to be enqueued, got: %v", enqueuedSales)
	}

	// Verify GitHub issue was created for Sale 100 failure
	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to query github issues: %v", err)
	}
	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected 1 reported issue for failed sale 100, got %d", len(resp.GetIssues()))
	}
	if resp.GetIssues()[0].GetTitle() != "Sale Adjustment Failure: adjustPrice on sale 100" {
		t.Errorf("unexpected issue title: %v", resp.GetIssues()[0].GetTitle())
	}

	// Verify user.LastSaleAdjust was updated
	savedUser, err := d.GetUser(ctx, user.GetAuth().GetToken())
	if err != nil {
		t.Fatalf("unable to load user: %v", err)
	}
	if savedUser.GetLastSaleAdjust() == 0 {
		t.Errorf("expected LastSaleAdjust to be updated")
	}
}

func TestAdjustSales_ExpectedLowPriceContinuesLoop(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	// Sale 1: Already reached low price (lowerBound == 0 with post-median conditions)
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:             100,
		ReleaseId:          1000,
		CurrentPrice:       &pbd.Price{Value: 50},
		MedianPrice:        &pbd.Price{Value: 200},
		LowPrice:           &pbd.Price{Value: 50},
		TimeAtMedian:       time.Now().Add(-time.Hour * 48).UnixNano(),
		LastPriceUpdate:    time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:          pbd.SaleStatus_FOR_SALE,
		SaleUpdateOverride: pb.SaleUpdateType_REDUCE_TO_MEDIAN_AND_THEN_LOW,
	})

	// Sale 2: Valid normal sale
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:             200,
		ReleaseId:          2000,
		CurrentPrice:       &pbd.Price{Value: 500},
		MedianPrice:        &pbd.Price{Value: 400},
		LastPriceUpdate:    time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:          pbd.SaleStatus_FOR_SALE,
		SaleUpdateOverride: pb.SaleUpdateType_REDUCE_TO_MEDIAN,
	})

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
	}
	d.SaveUser(ctx, user)

	config := &pb.SaleConfig{
		UpdateFrequencySeconds:       10,
		Reduction:                    50,
		PostMedianTime:               1,
		PostMedianReduction:          10,
		PostMedianReductionFrequency: 10,
		LowerBoundStrategy:           pb.LowerBoundStrategy_STATIC_LOW, // lowerBound = 0 -> "Already reached low price"
	}

	var enqueuedSales []int64
	enqueue := func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		if req.GetElement().GetUpdateSale() != nil {
			enqueuedSales = append(enqueuedSales, req.GetElement().GetUpdateSale().GetSaleId())
		}
		return &pb.EnqueueResponse{}, nil
	}

	err := b.AdjustSales(ctx, config, user, enqueue)
	if err != nil {
		t.Fatalf("expected AdjustSales to return nil (expected condition continues loop), got: %v", err)
	}

	// Verify Sale 200 was enqueued
	if len(enqueuedSales) != 1 || enqueuedSales[0] != 200 {
		t.Errorf("expected only Sale 200 to be enqueued, got: %v", enqueuedSales)
	}

	// Expected condition should NOT file a GitHub issue
	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to query github issues: %v", err)
	}
	if len(resp.GetIssues()) != 0 {
		t.Errorf("expected 0 issues for expected low price condition, got %d", len(resp.GetIssues()))
	}
}

func TestAdjustSales_InitialGetSalesFailureReturnsError(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	failingDB := &failingGetSalesDB{Database: d}
	b := GetBackgroundRunner(failingDB, "", "", "")

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
	}

	config := &pb.SaleConfig{
		UpdateFrequencySeconds: 10,
		Reduction:              50,
	}

	err := b.AdjustSales(ctx, config, user, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		return &pb.EnqueueResponse{}, nil
	})
	if err == nil {
		t.Fatalf("expected error from initial GetSales failure, got nil")
	}
}

func TestAdjustSales_ContextCancellationExitsCleanly(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          100,
		ReleaseId:       1000,
		CurrentPrice:    &pbd.Price{Value: 500},
		MedianPrice:     &pbd.Price{Value: 400},
		LastPriceUpdate: time.Now().Add(-time.Hour * 48).UnixNano(),
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
	}

	config := &pb.SaleConfig{
		UpdateFrequencySeconds: 10,
		Reduction:              50,
	}

	enqueued := false
	err := b.AdjustSales(ctx, config, user, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		enqueued = true
		return &pb.EnqueueResponse{}, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled error, got: %v", err)
	}
	if enqueued {
		t.Errorf("expected no sales to be enqueued when context is cancelled")
	}
}

func TestAddSale_PopulatesConditionAndSleeveCondition(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				ListingStrategy: pb.SaleConfig_LISTING_STRATEGY_MEDIAN,
			},
		},
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId:      1001,
			Id:              2001,
			Condition:       "Near Mint (NM or M-)",
			SleeveCondition: "Very Good Plus (VG+)",
		},
		MedianPrice: &pbd.Price{Value: 1500},
		LowPrice:    &pbd.Price{Value: 1000},
		HighPrice:   &pbd.Price{Value: 2000},
	})
	if err != nil {
		t.Fatalf("failed to save record: %v", err)
	}

	dc := &discogs.TestDiscogsClient{UserId: 123}
	err = b.AddSale(ctx, dc, 1001, &pbd.SaleParams{ReleaseId: 2001}, user)
	if err != nil {
		t.Fatalf("unexpected error from AddSale: %v", err)
	}

	sales, err := d.GetSales(ctx, 123)
	if err != nil {
		t.Fatalf("failed to get sales: %v", err)
	}
	if len(sales) == 0 {
		t.Fatalf("expected at least 1 sale saved, got 0")
	}

	sale, err := d.GetSale(ctx, 123, sales[0])
	if err != nil {
		t.Fatalf("failed to get sale: %v", err)
	}

	if sale.GetCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected Condition 'Near Mint (NM or M-)', got %q", sale.GetCondition())
	}
	if sale.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected SleeveCondition 'Very Good Plus (VG+)', got %q", sale.GetSleeveCondition())
	}
	if sale.GetMedianPrice().GetValue() != 1500 {
		t.Errorf("expected MedianPrice 1500, got %v", sale.GetMedianPrice().GetValue())
	}
	if sale.GetLowPrice().GetValue() != 1000 {
		t.Errorf("expected LowPrice 1000, got %v", sale.GetLowPrice().GetValue())
	}
	if sale.GetTimeCreated() == 0 {
		t.Errorf("expected TimeCreated to be set, got 0")
	}
	if sale.GetTimeRefreshed() == 0 {
		t.Errorf("expected TimeRefreshed to be set, got 0")
	}
	if sale.GetLastPriceUpdate() == 0 {
		t.Errorf("expected LastPriceUpdate to be set, got 0")
	}
}

func TestAddSale_FailsAndRaisesIssue_WhenConditionMissing(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.ghclient = ghClient

	user := &pb.StoredUser{
		User: &pbd.User{DiscogsUserId: 123},
		Auth: &pb.GramophileAuth{Token: "test_token"},
		Config: &pb.GramophileConfig{
			SaleConfig: &pb.SaleConfig{
				ListingStrategy: pb.SaleConfig_LISTING_STRATEGY_MEDIAN,
			},
		},
	}
	err := d.SaveUser(ctx, user)
	if err != nil {
		t.Fatalf("failed to save user: %v", err)
	}

	// Record with empty condition
	err = d.SaveRecord(ctx, 123, &pb.Record{
		Release: &pbd.Release{
			InstanceId:      1002,
			Id:              2002,
			Condition:       "",
			SleeveCondition: "Very Good Plus (VG+)",
		},
		MedianPrice: &pbd.Price{Value: 1500},
		LowPrice:    &pbd.Price{Value: 1000},
		HighPrice:   &pbd.Price{Value: 2000},
	})
	if err != nil {
		t.Fatalf("failed to save record: %v", err)
	}

	dc := &discogs.TestDiscogsClient{UserId: 123}
	err = b.AddSale(ctx, dc, 1002, &pbd.SaleParams{ReleaseId: 2002}, user)
	if err == nil {
		t.Fatalf("expected error from AddSale when condition is missing, got nil")
	}

	st, ok := status.FromError(err)
	if !ok || st.Code() != codes.FailedPrecondition {
		t.Errorf("expected FailedPrecondition status code, got %v", err)
	}

	resp, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("failed to get issues from test client: %v", err)
	}

	if len(resp.GetIssues()) != 1 {
		t.Fatalf("expected 1 issue to be created, got %d", len(resp.GetIssues()))
	}

	issue := resp.GetIssues()[0]
	expectedTitle := "Sale creation failed: record 1002 missing condition"
	if issue.GetTitle() != expectedTitle {
		t.Errorf("expected issue title %q, got %q", expectedTitle, issue.GetTitle())
	}
	if issue.GetRepo() != "gramophile" {
		t.Errorf("expected issue repo 'gramophile', got %q", issue.GetRepo())
	}
}

func TestSyncSales_PreservesConditionOnUpdate(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	userId := int32(123)

	// Existing sale 1: Empty condition, has sleeve condition -> should be backfilled from Discogs
	err := d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:          12345,
		ReleaseId:       100,
		Condition:       "",
		SleeveCondition: "Very Good (VG)",
		CurrentPrice:    &pbd.Price{Value: 1000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	if err != nil {
		t.Fatalf("failed to save sale 12345: %v", err)
	}

	// Existing sale 2: Non-empty condition and sleeve condition -> existing conditions should be preserved
	err = d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:          12346,
		ReleaseId:       101,
		Condition:       "Mint (M)",
		SleeveCondition: "Near Mint (NM)",
		CurrentPrice:    &pbd.Price{Value: 2000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	if err != nil {
		t.Fatalf("failed to save sale 12346: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: userId,
		Sales: []*pbd.SaleItem{
			{
				SaleId:    12345,
				ReleaseId: 100,
				Condition: "Very Good Plus (VG+)",
				Price:     &pbd.Price{Value: 1000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
			{
				SaleId:    12346,
				ReleaseId: 101,
				Condition: "Poor (P)",
				Price:     &pbd.Price{Value: 2000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
			{
				SaleId:    12347,
				ReleaseId: 102,
				Condition: "Near Mint (NM)",
				Price:     &pbd.Price{Value: 3000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
		},
	}

	b := GetBackgroundRunner(d, "", "", "")
	_, _, err = b.SyncSales(ctx, di, 1, time.Now().UnixNano(), 0)
	if err != nil {
		t.Fatalf("unexpected error from SyncSales: %v", err)
	}

	// Verify sale 12345: Condition backfilled, SleeveCondition preserved
	sale1, err := d.GetSale(ctx, userId, 12345)
	if err != nil {
		t.Fatalf("failed to get sale 12345: %v", err)
	}
	if sale1.GetCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected sale 12345 Condition to be backfilled to 'Very Good Plus (VG+)', got %q", sale1.GetCondition())
	}
	if sale1.GetSleeveCondition() != "Very Good (VG)" {
		t.Errorf("expected sale 12345 SleeveCondition to be preserved as 'Very Good (VG)', got %q", sale1.GetSleeveCondition())
	}

	// Verify sale 12346: Existing Condition and SleeveCondition preserved
	sale2, err := d.GetSale(ctx, userId, 12346)
	if err != nil {
		t.Fatalf("failed to get sale 12346: %v", err)
	}
	if sale2.GetCondition() != "Mint (M)" {
		t.Errorf("expected sale 12346 Condition to be preserved as 'Mint (M)', got %q", sale2.GetCondition())
	}
	if sale2.GetSleeveCondition() != "Near Mint (NM)" {
		t.Errorf("expected sale 12346 SleeveCondition to be preserved as 'Near Mint (NM)', got %q", sale2.GetSleeveCondition())
	}

	// Verify sale 12347: Newly discovered sale populated with condition
	sale3, err := d.GetSale(ctx, userId, 12347)
	if err != nil {
		t.Fatalf("failed to get sale 12347: %v", err)
	}
	if sale3.GetCondition() != "Near Mint (NM)" {
		t.Errorf("expected sale 12347 Condition to be 'Near Mint (NM)', got %q", sale3.GetCondition())
	}
}

func TestHardLink_PropagatesConditionFromRecord(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}
	record := &pb.Record{
		Release: &pbd.Release{
			Id:              100,
			InstanceId:      1000,
			Condition:       "Mint (M)",
			SleeveCondition: "Near Mint (NM or M-)",
		},
	}
	sale := &pb.SaleInfo{
		SaleId:          50,
		ReleaseId:       100,
		SaleState:       pbd.SaleStatus_FOR_SALE,
		Condition:       "Very Good Plus (VG+)",
		SleeveCondition: "Very Good (VG)",
	}

	err := d.SaveSale(ctx, 123, sale)
	if err != nil {
		t.Fatalf("unable to save initial sale: %v", err)
	}

	err = b.HardLink(ctx, user, []*pb.Record{record}, []*pb.SaleInfo{sale})
	if err != nil {
		t.Fatalf("HardLink failed: %v", err)
	}

	savedSale, err := d.GetSale(ctx, 123, 50)
	if err != nil {
		t.Fatalf("unable to get updated sale: %v", err)
	}

	if savedSale.GetCondition() != "Mint (M)" {
		t.Errorf("expected condition 'Mint (M)', got '%v'", savedSale.GetCondition())
	}
	if savedSale.GetSleeveCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected sleeve condition 'Near Mint (NM or M-)', got '%v'", savedSale.GetSleeveCondition())
	}
}

func TestHardLink_RaisesIssueForUnmatchedActiveSale(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)
	b := GetBackgroundRunner(d, "", "", "")
	ghClient := ghbclient.GetTestClient()
	b.SetGHClient(ghClient)

	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}
	records := []*pb.Record{
		{
			Release: &pbd.Release{Id: 100, InstanceId: 1000},
		},
	}
	sales := []*pb.SaleInfo{
		{
			SaleId:    50,
			ReleaseId: 100,
			SaleState: pbd.SaleStatus_FOR_SALE,
		},
		{
			SaleId:    999,
			ReleaseId: 200,
			SaleState: pbd.SaleStatus_FOR_SALE,
		},
		{
			SaleId:    888,
			ReleaseId: 300,
			SaleState: pbd.SaleStatus_SOLD,
		},
	}

	err := b.HardLink(ctx, user, records, sales)
	if err != nil {
		t.Fatalf("HardLink failed: %v", err)
	}

	issues, err := ghClient.GetIssues(ctx, &ghbpb.GetIssuesRequest{})
	if err != nil {
		t.Fatalf("unable to get issues: %v", err)
	}

	if len(issues.GetIssues()) != 1 {
		t.Fatalf("expected 1 issue to be raised, got %v: %v", len(issues.GetIssues()), issues.GetIssues())
	}

	expectedTitle := "Active Discogs sale 999 has no matching local record"
	if issues.GetIssues()[0].GetTitle() != expectedTitle {
		t.Errorf("expected issue title '%v', got '%v'", expectedTitle, issues.GetIssues()[0].GetTitle())
	}
}

func TestAdjustSales_PropagatesSleeveCondition(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	var recordedUpdate *pb.UpdateSale

	ctx := context.Background()
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		ReleaseId:       123,
		SaleId:          12345,
		CurrentPrice:    &pbd.Price{Value: 300},
		MedianPrice:     &pbd.Price{Value: 200},
		LowPrice:        &pbd.Price{Value: 50},
		Condition:       "Very Good Plus (VG+)",
		SleeveCondition: "Generic",
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})

	b := GetBackgroundRunner(d, "", "", "")

	err := b.AdjustSales(ctx, &pb.SaleConfig{
		UpdateType: pb.SaleUpdateType_REDUCE_TO_MEDIAN,
	}, &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}, Auth: &pb.GramophileAuth{Token: "123"}}, func(ctx context.Context, req *pb.EnqueueRequest) (*pb.EnqueueResponse, error) {
		recordedUpdate = req.GetElement().GetUpdateSale()
		return &pb.EnqueueResponse{}, nil
	})

	if err != nil {
		t.Fatalf("AdjustSales failed: %v", err)
	}

	if recordedUpdate == nil {
		t.Fatalf("expected UpdateSale to be enqueued, got nil")
	}

	if recordedUpdate.GetCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected Condition to be 'Very Good Plus (VG+)', got '%v'", recordedUpdate.GetCondition())
	}
	if recordedUpdate.GetSleeveCondition() != "Generic" {
		t.Errorf("expected SleeveCondition to be 'Generic', got '%v'", recordedUpdate.GetSleeveCondition())
	}
}

func TestProcessUpdateSale_ConditionGuardAndPreservation(t *testing.T) {
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	ctx := context.Background()
	user := &pb.StoredUser{User: &pbd.User{DiscogsUserId: 123}}

	// Case 1: Empty condition should be skipped silently without updating sale
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1001,
		ReleaseId:       2001,
		Condition:       "",
		SleeveCondition: "Very Good Plus (VG+)",
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	dc := &discogs.TestDiscogsClient{
		UserId: 123,
		Sales: []*pbd.SaleItem{
			{SaleId: 1001, ReleaseId: 2001, Price: &pbd.Price{Value: 5000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
		},
	}
	b := GetBackgroundRunner(d, "", "", "")

	err := b.ProcessUpdateSale(ctx, dc, user, &pb.UpdateSale{
		SaleId:          1001,
		ReleaseId:       2001,
		NewPrice:        4900,
		Condition:       "",
		SleeveCondition: "Very Good Plus (VG+)",
	})
	if err != nil {
		t.Fatalf("ProcessUpdateSale failed on empty condition: %v", err)
	}
	// Sale in DB should not have price updated because condition was empty
	sale1001, err := d.GetSale(ctx, 123, 1001)
	if err != nil {
		t.Fatalf("unable to get sale: %v", err)
	}
	if sale1001.GetCurrentPrice().GetValue() != 5000 {
		t.Errorf("expected sale price to remain 5000 when skipped, got: %v", sale1001.GetCurrentPrice().GetValue())
	}

	// Case 2: Populated condition updates price and preserves conditions
	d.SaveSale(ctx, 123, &pb.SaleInfo{
		SaleId:          1002,
		ReleaseId:       2002,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		CurrentPrice:    &pbd.Price{Value: 5000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	dc = &discogs.TestDiscogsClient{
		UserId: 123,
		Sales: []*pbd.SaleItem{
			{SaleId: 1002, ReleaseId: 2002, Price: &pbd.Price{Value: 5000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
		},
	}

	err = b.ProcessUpdateSale(ctx, dc, user, &pb.UpdateSale{
		SaleId:          1002,
		ReleaseId:       2002,
		NewPrice:        4800,
		Condition:       "Near Mint (NM or M-)",
		SleeveCondition: "Very Good Plus (VG+)",
		Motivation:      "Automated Reduction",
	})
	if err != nil {
		t.Fatalf("ProcessUpdateSale failed on valid condition: %v", err)
	}

	sale1002, err := d.GetSale(ctx, 123, 1002)
	if err != nil {
		t.Fatalf("unable to get sale 1002: %v", err)
	}
	if sale1002.GetCurrentPrice().GetValue() != 4800 {
		t.Errorf("expected updated price 4800, got: %v", sale1002.GetCurrentPrice().GetValue())
	}
	if sale1002.GetCondition() != "Near Mint (NM or M-)" {
		t.Errorf("expected Condition to be preserved as 'Near Mint (NM or M-)', got '%v'", sale1002.GetCondition())
	}
	if sale1002.GetSleeveCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected SleeveCondition to be preserved as 'Very Good Plus (VG+)', got '%v'", sale1002.GetSleeveCondition())
	}
}

func TestSyncSales_Incremental_EarlyTermination(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	userId := int32(123)
	lastSaleRefresh := int64(2000)

	// Existing sale in DB with ListedDate <= lastSaleRefresh (1000 <= 2000)
	err := d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:       1001,
		ReleaseId:    2001,
		Condition:    "Mint (M)",
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
		SaleState:    pbd.SaleStatus_FOR_SALE,
		ListedDate:   1000,
	})
	if err != nil {
		t.Fatalf("failed to save existing sale: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: userId,
		Sales: []*pbd.SaleItem{
			{SaleId: 1002, ReleaseId: 2002, Price: &pbd.Price{Value: 3000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
			{SaleId: 1001, ReleaseId: 2001, Price: &pbd.Price{Value: 1000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
			{SaleId: 1003, ReleaseId: 2003, Price: &pbd.Price{Value: 4000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
		},
	}

	b := GetBackgroundRunner(d, "", "", "")
	pagination, earlyTerminated, err := b.SyncSales(ctx, di, 1, 999, lastSaleRefresh)
	if err != nil {
		t.Fatalf("unexpected error from SyncSales: %v", err)
	}
	if pagination == nil {
		t.Fatalf("expected non-nil pagination")
	}
	if !earlyTerminated {
		t.Errorf("expected earlyTerminated to be true, got false")
	}

	// Verify sale 1002 (new sale before early termination) was saved
	_, err = d.GetSale(ctx, userId, 1002)
	if err != nil {
		t.Errorf("expected sale 1002 to be saved in DB, got error: %v", err)
	}

	// Verify sale 1001 (existing sale that triggered early termination) was updated
	_, err = d.GetSale(ctx, userId, 1001)
	if err != nil {
		t.Errorf("expected sale 1001 to be retrieved from DB, got error: %v", err)
	}

	// Verify sale 1003 (subsequent sale after early termination) was NOT processed/saved
	_, err = d.GetSale(ctx, userId, 1003)
	if status.Code(err) != codes.NotFound {
		t.Errorf("expected sale 1003 to NOT be saved in DB (NotFound), got err: %v", err)
	}
}

func TestSyncSales_ColdStart_FullPagination(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	userId := int32(123)

	// Existing sale in DB
	err := d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:       1001,
		ReleaseId:    2001,
		Condition:    "Mint (M)",
		CurrentPrice: &pbd.Price{Value: 1000, Currency: "USD"},
		SaleState:    pbd.SaleStatus_FOR_SALE,
		ListedDate:   1000,
	})
	if err != nil {
		t.Fatalf("failed to save existing sale: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: userId,
		Sales: []*pbd.SaleItem{
			{SaleId: 1002, ReleaseId: 2002, Price: &pbd.Price{Value: 3000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
			{SaleId: 1001, ReleaseId: 2001, Price: &pbd.Price{Value: 1000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
			{SaleId: 1003, ReleaseId: 2003, Price: &pbd.Price{Value: 4000, Currency: "USD"}, Status: pbd.SaleStatus_FOR_SALE},
		},
	}

	b := GetBackgroundRunner(d, "", "", "")
	// lastSaleRefresh == 0 (cold start / baseline)
	pagination, earlyTerminated, err := b.SyncSales(ctx, di, 1, 999, 0)
	if err != nil {
		t.Fatalf("unexpected error from SyncSales: %v", err)
	}
	if pagination == nil {
		t.Fatalf("expected non-nil pagination")
	}
	if earlyTerminated {
		t.Errorf("expected earlyTerminated to be false on cold start, got true")
	}

	// Verify all 3 sales are in DB
	for _, saleId := range []int64{1001, 1002, 1003} {
		_, err = d.GetSale(ctx, userId, saleId)
		if err != nil {
			t.Errorf("expected sale %v to be saved in DB, got error: %v", saleId, err)
		}
	}
}

func TestSyncSales_MetadataPreservation(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	userId := int32(123)

	// Existing sale 1: Empty condition, has sleeve condition -> condition backfilled from Discogs
	err := d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:          12345,
		ReleaseId:       100,
		Condition:       "",
		SleeveCondition: "Very Good (VG)",
		CurrentPrice:    &pbd.Price{Value: 1000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	if err != nil {
		t.Fatalf("failed to save sale 12345: %v", err)
	}

	// Existing sale 2: Non-empty condition and sleeve condition -> existing conditions preserved
	err = d.SaveSale(ctx, userId, &pb.SaleInfo{
		SaleId:          12346,
		ReleaseId:       101,
		Condition:       "Mint (M)",
		SleeveCondition: "Near Mint (NM)",
		CurrentPrice:    &pbd.Price{Value: 2000, Currency: "USD"},
		SaleState:       pbd.SaleStatus_FOR_SALE,
	})
	if err != nil {
		t.Fatalf("failed to save sale 12346: %v", err)
	}

	di := &discogs.TestDiscogsClient{
		UserId: userId,
		Sales: []*pbd.SaleItem{
			{
				SaleId:    12345,
				ReleaseId: 100,
				Condition: "Very Good Plus (VG+)",
				Price:     &pbd.Price{Value: 1000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
			{
				SaleId:    12346,
				ReleaseId: 101,
				Condition: "Poor (P)",
				Price:     &pbd.Price{Value: 2000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
			{
				SaleId:    12347,
				ReleaseId: 102,
				Condition: "Near Mint (NM)",
				Price:     &pbd.Price{Value: 3000, Currency: "USD"},
				Status:    pbd.SaleStatus_FOR_SALE,
			},
		},
	}

	b := GetBackgroundRunner(d, "", "", "")
	_, earlyTerminated, err := b.SyncSales(ctx, di, 1, time.Now().UnixNano(), 0)
	if err != nil {
		t.Fatalf("unexpected error from SyncSales: %v", err)
	}
	if earlyTerminated {
		t.Errorf("expected earlyTerminated to be false, got true")
	}

	// Verify sale 12345: Condition backfilled, SleeveCondition preserved
	sale1, err := d.GetSale(ctx, userId, 12345)
	if err != nil {
		t.Fatalf("failed to get sale 12345: %v", err)
	}
	if sale1.GetCondition() != "Very Good Plus (VG+)" {
		t.Errorf("expected sale 12345 Condition to be backfilled to 'Very Good Plus (VG+)', got %q", sale1.GetCondition())
	}
	if sale1.GetSleeveCondition() != "Very Good (VG)" {
		t.Errorf("expected sale 12345 SleeveCondition to be preserved as 'Very Good (VG)', got %q", sale1.GetSleeveCondition())
	}

	// Verify sale 12346: Existing Condition and SleeveCondition preserved
	sale2, err := d.GetSale(ctx, userId, 12346)
	if err != nil {
		t.Fatalf("failed to get sale 12346: %v", err)
	}
	if sale2.GetCondition() != "Mint (M)" {
		t.Errorf("expected sale 12346 Condition to be preserved as 'Mint (M)', got %q", sale2.GetCondition())
	}
	if sale2.GetSleeveCondition() != "Near Mint (NM)" {
		t.Errorf("expected sale 12346 SleeveCondition to be preserved as 'Near Mint (NM)', got %q", sale2.GetSleeveCondition())
	}

	// Verify sale 12347: Newly discovered sale populated with condition and ListedDate
	sale3, err := d.GetSale(ctx, userId, 12347)
	if err != nil {
		t.Fatalf("failed to get sale 12347: %v", err)
	}
	if sale3.GetCondition() != "Near Mint (NM)" {
		t.Errorf("expected sale 12347 Condition to be 'Near Mint (NM)', got %q", sale3.GetCondition())
	}
	if sale3.GetListedDate() == 0 {
		t.Errorf("expected sale 12347 ListedDate to be populated (>0), got %v", sale3.GetListedDate())
	}
}

func TestSyncSales_EmptyInventory(t *testing.T) {
	ctx := context.Background()
	pstore := pstore_client.GetTestClient()
	d := db.NewTestDB(pstore)

	userId := int32(123)

	di := &discogs.TestDiscogsClient{
		UserId: userId,
		Sales:  []*pbd.SaleItem{},
	}

	b := GetBackgroundRunner(d, "", "", "")
	pagination, earlyTerminated, err := b.SyncSales(ctx, di, 1, 999, 1000)
	if err != nil {
		t.Fatalf("unexpected error from SyncSales: %v", err)
	}
	if pagination == nil {
		t.Fatalf("expected non-nil pagination")
	}
	if earlyTerminated {
		t.Errorf("expected earlyTerminated to be false on empty inventory, got true")
	}
}



