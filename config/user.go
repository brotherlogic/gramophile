package config

import (
	"context"
	"fmt"
	"log"
	"strings"

	pbd "github.com/brotherlogic/discogs/proto"
	pb "github.com/brotherlogic/gramophile/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/reflect/protoreflect"
)

var (
	standardPaths []string = []string{}
	betaPaths     []string = []string{"cleaning_config.cleaning"}
)

func setToDefault(c *pb.GramophileConfig, path string) error {
	fields := strings.Split(path, ".")
	return setToDefaultArr(c.ProtoReflect(), fields)
}

func setToDefaultArr(c protoreflect.Message, fields []string) error {
	log.Printf("VALIDATING %v against %v", c, fields)
	if len(fields) == 1 {
		pfields := c.Descriptor().Fields()
		for i := 0; i < pfields.Len(); i++ {
			if pfields.Get(i).TextName() == fields[0] {
				if pfields.Get(i).Kind() == protoreflect.BoolKind {
					if c.Get(pfields.Get(i)).Bool() {
						c.Set(pfields.Get(i), protoreflect.ValueOfBool(false))
					}
					return nil
				} else if pfields.Get(i).Kind() == protoreflect.EnumKind {
					if c.Get(pfields.Get(i)).Enum() != 0 {
						c.Set(pfields.Get(i), protoreflect.ValueOfEnum(0))
					}
					return nil
				} else {
					return fmt.Errorf("Can only set bools or enums")
				}
			}
		}
		return fmt.Errorf("Unable to locate %v in proto %v", fields[0], c.Type())
	}

	pfields := c.Descriptor().Fields()
	for i := 0; i < pfields.Len(); i++ {
		if pfields.Get(i).TextName() == fields[0] {
			return setToDefaultArr(c.Get(pfields.Get(i)).Message(), fields[1:])
		}
	}

	// Shouldn't get here
	return fmt.Errorf("Unable to locate %v in proto %v", fields[0], c.Type())
}

type userConfig struct{}

func (*userConfig) GetClassification(c *pb.GramophileConfig) []*pb.Classifier {
	return []*pb.Classifier{}
}

func (*userConfig) PostProcess(c *pb.GramophileConfig) (*pb.GramophileConfig, error) {
	if c.GetUserConfig().GetUserLevel() == pb.UserConfig_USER_LEVEL_OMNIPOTENT {
		return c, nil
	}

	// Apply all the beta reductions
	for _, rule := range betaPaths {
		err := setToDefault(c, rule)
		if err != nil {
			return nil, err
		}
	}
	if c.GetUserConfig().GetUserLevel() == pb.UserConfig_USER_LEVEL_BETA {
		return c, nil
	}

	for _, rule := range standardPaths {
		setToDefault(c, rule)
	}
	return c, nil
}

func (*userConfig) Validate(ctx context.Context, fields []*pbd.Field, u *pb.StoredUser) error {
	if u.GetConfig().GetUserConfig().GetUserLevel() == pb.UserConfig_USER_LEVEL_OMNIPOTENT {
		if u.GetUser().GetDiscogsUserId() != 150295 {
			return status.Errorf(codes.FailedPrecondition, "You are not allowed to set the user level to omnipotent")
		}
	}
	return nil
}
