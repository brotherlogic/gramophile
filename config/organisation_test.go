package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestOrganisation_FailOnWidth(t *testing.T) {
	c := &pb.GramophileConfig{OrganisationConfig: &pb.OrganisationConfig{
		Organisations: []*pb.Organisation{
			{
				Name:    "testing",
				Density: pb.Density_WIDTH,
			},
		},
	}}

	_, _, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}}, c)
	if err == nil {
		t.Errorf("Should have failed because of missing width config")
	}
}

func TestOrganisation_Success(t *testing.T) {
	c := &pb.GramophileConfig{
		WidthConfig: &pb.WidthConfig{Mandate: pb.Mandate_REQUIRED},
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:    "testing",
					Density: pb.Density_WIDTH,
				},
			},
		}}

	_, _, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}}, c)
	if err == nil {
		t.Errorf("Should have failed because of missing width config")
	}
}
