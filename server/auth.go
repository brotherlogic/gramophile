package server

import (
	"reflect"

	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	featureGating = map[reflect.Type]pb.UserConfig_UserLevel{
		reflect.TypeOf(&pb.CleaningConfig{}):   pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.ListenConfig{}):     pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.WidthConfig{}):      pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.OrganisationConfig{}): pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.WeightConfig{}):     pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.GoalFolderConfig{}): pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.SleeveConfig{}):     pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.ArrivedConfig{}):    pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.SaleConfig{}):       pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.KeepConfig{}):       pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.WantsConfig{}):      pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.PrintMoveConfig{}):  pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.MintUpConfig{}):     pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.WantslistConfig{}):  pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.ScoreConfig{}):      pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.ClassificationConfig{}): pb.UserConfig_USER_LEVEL_STANDARD,
		reflect.TypeOf(&pb.MovingConfig{}):     pb.UserConfig_USER_LEVEL_BETA,
		reflect.TypeOf(&pb.AddConfig{}):        pb.UserConfig_USER_LEVEL_STANDARD,
	}
)

func CheckAccess(userLevel pb.UserConfig_UserLevel, configField interface{}) error {
	t := reflect.TypeOf(configField)
	if requiredLevel, ok := featureGating[t]; ok {
		if userLevel < requiredLevel {
			return status.Errorf(codes.PermissionDenied, "Feature requires %v, but user is %v", requiredLevel, userLevel)
		}
	}
	return nil
}
