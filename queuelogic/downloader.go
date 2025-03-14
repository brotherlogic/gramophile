package queuelogic

import (
	"context"

	scraper_client "github.com/brotherlogic/scraper/client"
	pb "github.com/brotherlogic/scraper/proto"
)

type DownloaderBridge struct {
	scraper scraper_client.ScraperClient
}

func (d *DownloaderBridge) Download(ctx context.Context, url string) (string, error) {
	req, err := d.scraper.Scrape(ctx, &pb.ScrapeRequest{
		Url: url,
	})
	if err != nil {
		return "", err
	}
	return req.GetBody(), nil
}
