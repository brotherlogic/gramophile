package server

import (
	"context"
	"os"

	"github.com/brotherlogic/discogs"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {
	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))
	url, secret, err := d.GetLoginURL()
	return &pb.GetURLResponse{URL: url, Secret: secret}, err
}
