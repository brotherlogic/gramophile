package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pb "github.com/brotherlogic/gramophile/proto"
)

var (
	runResult = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_result",
		Help: "The length of the working queue I think yes",
	}, []string{"result"})
	loopTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_loop_time",
		Help: "The length of the working queue I think yes",
	}, []string{"result"})

	collectionSize = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_collection",
	}, []string{"folder"})
	salesByYear = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_sales",
	}, []string{"year"})
	salesByState = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_sales_by_state",
		Help: "The name by the given state",
	}, []string{"state"})
	saleUpdates = promauto.NewHistogram(prometheus.HistogramOpts{
		Name: "gramophile_overseer_sale_updates",
		Help: "The name by the given state",
	})

	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")
)

func runLoop(ctx context.Context) error {
	// Get all the users
	conn, err := grpc.NewClient("gramophile.gramophile:8080", grpc.WithInsecure())
	if err != nil {
		return err
	}
	client := pb.NewGramophileEServiceClient(conn)

	stats, err := client.GetStats(ctx, &pb.GetStatsRequest{})
	if err != nil {
		return err
	}

	collectionSize.Reset()
	for folder, count := range stats.GetCollectionStats().GetFolderToCount() {
		collectionSize.With(prometheus.Labels{"folder": fmt.Sprintf("%v", folder)}).Set(float64(count))
	}

	salesByYear.Reset()
	for year, total := range stats.GetSaleStats().GetYearTotals() {
		salesByYear.With(prometheus.Labels{"year": fmt.Sprintf("%v", year)}).Set(float64(total))
	}

	salesByState.Reset()
	for cat, total := range stats.GetSaleStats().GetStateCount() {
		salesByState.With(prometheus.Labels{"state": cat}).Set(float64(total))
	}

	return nil
}

func main() {
	//Run a loop every minute
	go func() {
		for {
			t1 := time.Now()
			mContext := metadata.AppendToOutgoingContext(context.Background(), "auth-token", os.Getenv("token"))
			ctx, cancel := context.WithTimeout(mContext, time.Minute)
			err := runLoop(ctx)
			log.Printf("Ran loop: %v", err)
			cancel()
			runResult.With(prometheus.Labels{"result": fmt.Sprintf("%v", status.Code(err))}).Set(float64(time.Now().Unix()))
			loopTime.With(prometheus.Labels{"result": fmt.Sprintf("%v", status.Code(err))}).Set(float64(time.Since(t1).Milliseconds()))
			time.Sleep(time.Minute)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
	if err != nil {
		log.Printf("gramophile is unable to serve metrics: %v", err)
	}
}
