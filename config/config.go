package config

import (
	"crypto/md5"
	"encoding/hex"

	pb "github.com/brotherlogic/gramophile/proto"

	"google.golang.org/protobuf/proto"
)

func Hash(c *pb.GramophileConfig) string {
	bytes, _ := proto.Marshal(c)
	hash := md5.Sum(bytes)
	return hex.EncodeToString(hash[:])
}
