package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestSleeveFailed_NoField(t *testing.T) {
	c := &pb.GramophileConfig{SleeveConfig: &pb.SleeveConfig{Mandate: pb.Mandate_RECOMMENDED,
		AllowedSleeves: []*pb.Sleeve{{Name: "test"}}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, c)
	if err == nil {
		t.Errorf("This should have failed but did not")
	}
}

func TestSleeveFailed_NoSleeves(t *testing.T) {
	c := &pb.GramophileConfig{SleeveConfig: &pb.SleeveConfig{Mandate: pb.Mandate_RECOMMENDED}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Sleeve", Id: 1}}, c)
	if err == nil {
		t.Errorf("This should have failed but did not")
	}
}

func TestSleeveSuccess(t *testing.T) {
	c := &pb.GramophileConfig{SleeveConfig: &pb.SleeveConfig{Mandate: pb.Mandate_RECOMMENDED,
		AllowedSleeves: []*pb.Sleeve{{Name: "test"}}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Sleeve", Id: 1}}, c)
	if err != nil {
		t.Errorf("validate weight raised an error: %v", err)
	}
}
