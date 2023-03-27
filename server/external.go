package server

import (
	"context"
	"log"
	"os"

	"github.com/brotherlogic/discogs"

	pb "github.com/brotherlogic/gramophile/proto"
)

func (s *Server) GetURL(ctx context.Context, req *pb.GetURLRequest) (*pb.GetURLResponse, error) {

	log.Printf("%v and %v and %v", os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))

	d := discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK"))
	url, _, err := d.GetLoginURL()
	return &pb.GetURLResponse{URL: url}, err
}
