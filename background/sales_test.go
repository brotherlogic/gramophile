package background

import (
	"sort"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
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
