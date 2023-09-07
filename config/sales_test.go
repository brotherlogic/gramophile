package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSales_FailedNoField(t *testing.T) {
	c := &pb.GramophileConfig{SaleConfig: &pb.SaleConfig{Mandate: pb.Mandate_RECOMMENDED}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("This should have failed but did not")
	}
}

func TestSales_Success(t *testing.T) {
	c := &pb.GramophileConfig{SaleConfig: &pb.SaleConfig{Mandate: pb.Mandate_RECOMMENDED}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "LastSaleUpdate", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate sale config raised an error: %v", err)
	}
}

func TestSales_AddsMoves(t *testing.T) {
	c := &pb.GramophileConfig{
		SaleConfig: &pb.SaleConfig{Mandate: pb.Mandate_RECOMMENDED}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "LastSaleUpdate", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate sale config raised an error: %v", err)
	}

	/*c, err := ExpandConfig(context.Background(), c)
	if err != nil {
		t.Fatalf("Expand out the config: %v", err)
	}

	if len(c.GetMoveConfig().GetMoves()) != 2 {
		t.Errorf("Moves were not created: %v", c.GetMoveConfig())
	}*/
}
