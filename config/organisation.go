package config

import (
	"context"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type org struct{}

func (*org) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*org) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	return c, nil
}

func (*org) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {

	// Raise an error if any org relies on width being set
	hasWidthMandate := u.GetConfig().GetWidthConfig().GetEnabled() == pb.Enabled_ENABLED_ENABLED
	
	orgNames := make(map[string]bool)
	folderMapped := make(map[int32]string)

	for _, org := range u.GetConfig().GetOrganisationConfig().GetOrganisations() {
		if org.GetDensity() == pb.Density_WIDTH && !hasWidthMandate {
			return status.Errorf(codes.FailedPrecondition, "%v requires width mandate", org.GetName())
		}
		
		if orgNames[org.GetName()] {
			return status.Errorf(codes.AlreadyExists, "duplicate organisation name: %v", org.GetName())
		}
		orgNames[org.GetName()] = true

		spaceNames := make(map[string]bool)
		for _, space := range org.GetSpaces() {
			if space.GetName() == "" {
				return status.Errorf(codes.InvalidArgument, "space name cannot be blank in organisation: %v", org.GetName())
			}
			if spaceNames[space.GetName()] {
				return status.Errorf(codes.InvalidArgument, "duplicate space name %v in organisation: %v", space.GetName(), org.GetName())
			}
			spaceNames[space.GetName()] = true
		}

		for _, fs := range org.GetFoldersets() {
			if existingOrg, ok := folderMapped[fs.GetFolder()]; ok {
				return status.Errorf(codes.FailedPrecondition, "folder %v is mapped to multiple organisations: %v and %v", fs.GetFolder(), existingOrg, org.GetName())
			}
			folderMapped[fs.GetFolder()] = org.GetName()
		}
	}

	return nil
}
