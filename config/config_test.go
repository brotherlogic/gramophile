package config

import (
	"context"
	"testing"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
)

func TestValidation(t *testing.T) {
	config := &pb.StoredUser{Config: &pb.GramophileConfig{
		CleaningConfig: &pb.CleaningConfig{
			CleaningGapInSeconds: 5,
			CleaningGapInPlays:   2,
		},
	}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, config)
	if err == nil {
		t.Errorf("Config was validated: %v", config)
	}
}

func TestValidationOverlappingFolders(t *testing.T) {
	config := &pb.StoredUser{Config: &pb.GramophileConfig{
		OrganisationConfig: &pb.OrganisationConfig{
			Organisations: []*pb.Organisation{
				{
					Name: "TestOrg",
					Foldersets: []*pb.FolderSet{
						{Folder: 123},
						{Folder: 123},
					},
				},
			},
		},
	}}

	_, err := ValidateConfig(context.Background(), &pb.StoredUser{}, []*pbd.Field{}, config)
	if err == nil {
		t.Errorf("Config was validated despite overlapping folders: %v", config)
	}
}
