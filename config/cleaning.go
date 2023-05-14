package config

import (
	"fmt"

	pb "github.com/brotherlogic/gramophile/proto"
)

type cleaning struct{}

func (*cleaning) Validate(c *pb.GramophileConfig) error {
	if c.GetCleaningConfig().GetCleaningGapInPlays() > 0 && c.GetCleaningConfig().GetCleaningGapInSeconds() > 0 {
		return fmt.Errorf("You must set one of plays or seconds, not both")
	}
	return nil
}
