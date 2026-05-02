package server

import (
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GetAccessLevel(configField interface{}) pb.UserConfig_UserLevel {
	switch configField.(type) {
	case *pb.CleaningConfig,
		*pb.ListenConfig,
		*pb.WidthConfig,
		*pb.OrganisationConfig,
		*pb.WeightConfig,
		*pb.GoalFolderConfig,
		*pb.SleeveConfig,
		*pb.ArrivedConfig,
		*pb.SaleConfig,
		*pb.KeepConfig,
		*pb.WantsConfig,
		*pb.PrintMoveConfig,
		*pb.MintUpConfig,
		*pb.WantslistConfig,
		*pb.ScoreConfig,
		*pb.ClassificationConfig,
		*pb.AddConfig,
		*pb.UserConfig:
		return pb.UserConfig_USER_LEVEL_STANDARD
	case *pb.MovingConfig:
		return pb.UserConfig_USER_LEVEL_BETA
	default:
		// Security: Deny-by-default: require OMNIPOTENT (SUPERUSER) access for unrecognized config
		return pb.UserConfig_USER_LEVEL_OMNIPOTENT
	}
}

func CheckAccess(userLevel pb.UserConfig_UserLevel, configField interface{}) error {
	requiredLevel := GetAccessLevel(configField)
	if userLevel < requiredLevel {
		return status.Errorf(codes.PermissionDenied, "Feature requires %v, but user is %v", requiredLevel, userLevel)
	}
	return nil
}
