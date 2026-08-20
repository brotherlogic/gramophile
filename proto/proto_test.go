package proto

import (
	"testing"
)

func TestLocationRecord(t *testing.T) {
	loc := &Location{
		Record: "Test Artist - Test Title",
	}

	if loc.GetRecord() != "Test Artist - Test Title" {
		t.Errorf("Expected record 'Test Artist - Test Title', got %v", loc.GetRecord())
	}
}

func TestAdjustSalesProto(t *testing.T) {
	adjust := &AdjustSales{}
	elem := &QueueElement{
		Entry: &QueueElement_AdjustSales{
			AdjustSales: adjust,
		},
	}
	if elem.GetAdjustSales() == nil {
		t.Errorf("Expected non-nil AdjustSales in QueueElement")
	}

	user := &StoredUser{
		LastSaleAdjust: 123456789,
	}
	if user.GetLastSaleAdjust() != 123456789 {
		t.Errorf("Expected LastSaleAdjust to be 123456789, got %v", user.GetLastSaleAdjust())
	}
}

func TestSaleInfoSleeveCondition(t *testing.T) {
	sale := &SaleInfo{
		SleeveCondition: "Mint (M)",
	}
	if sale.GetSleeveCondition() != "Mint (M)" {
		t.Errorf("Expected sleeve condition 'Mint (M)', got %v", sale.GetSleeveCondition())
	}
}

func TestUpdateSaleSleeveCondition(t *testing.T) {
	update := &UpdateSale{
		SleeveCondition: "Near Mint (NM or M-)",
	}
	if update.GetSleeveCondition() != "Near Mint (NM or M-)" {
		t.Errorf("Expected sleeve condition 'Near Mint (NM or M-)', got %v", update.GetSleeveCondition())
	}
}


