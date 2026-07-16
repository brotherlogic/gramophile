package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestOrganisation_FailOnWidth(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{OrganisationConfig: &pb.OrganisationConfig{
		Organisations: []*pb.Organisation{
			{
				Name:    "testing",
				Density: pb.Density_WIDTH,
			},
		},
	}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}}, c)
	if err == nil {
		t.Errorf("Should have failed because of missing width config")
	}
}

func TestOrganisation_Success(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{
		WidthConfig: &pb.WidthConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:    "testing",
					Density: pb.Density_WIDTH,
				},
			},
		}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}}, c)
	if err == nil {
		t.Errorf("Should have failed because of missing width config")
	}
}

func TestOrganisation_DuplicateName(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{
		WidthConfig: &pb.WidthConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:    "testing",
					Density: pb.Density_WIDTH,
				},
				{
					Name:    "testing",
					Density: pb.Density_WIDTH,
				},
			},
		}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}, {Name: "Width", Id: 2}}, c)
	if err == nil || status.Code(err) != codes.AlreadyExists {
		t.Errorf("Should have failed with AlreadyExists: %v", err)
	}
}

func TestOrganisation_OverlappingFolders(t *testing.T) {
	c := &pb.StoredUser{Config: &pb.GramophileConfig{
		WidthConfig: &pb.WidthConfig{Enabled: pb.Enabled_ENABLED_ENABLED},
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name:    "testing1",
					Density: pb.Density_WIDTH,
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
					},
				},
				{
					Name:    "testing2",
					Density: pb.Density_WIDTH,
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
					},
				},
			},
		}}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{{Name: "Arrived", Id: 1}, {Name: "Width", Id: 2}}, c)
	if err == nil || (status.Code(err) != codes.FailedPrecondition && status.Code(err) != codes.InvalidArgument) {
		t.Errorf("Should have failed with FailedPrecondition or InvalidArgument: %v", err)
	}
}
