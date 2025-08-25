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

	ghb "github.com/brotherlogic/githubridge/client"

	pbgh "github.com/brotherlogic/githubridge/proto"
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
	oldestSale = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_overseer_oldest_sale",
	})
	oldestSaleId = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_overseer_oldest_sale_id",
	})
	erdMissing = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_overseer_erd_missing",
		Help: "The number of records without ERD",
	})
	wants = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_overseer_total_wants",
		Help: "The number of wants",
	})
	syncedWants = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "gramophile_overseer_synced_wants",
		Help: "The number of wants",
	})

	launchTotal = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_launch_total",
	}, []string{"version"})
	launchDone = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "gramophile_overseer_launch_done",
	}, []string{"version"})

	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")
)

func getLaunchBug(ctx context.Context, user, repo string, number int32) (*pbgh.GetIssueResponse, error) {
	ghbclient, err := ghb.GetClientInternal()
	if err != nil {
		return nil, err
	}

	issue, err := ghbclient.GetIssue(ctx, &pbgh.GetIssueRequest{
		User: user,
		Repo: repo,
		Id:   number,
	})

	if err != nil {
		return nil, err
	}

	return issue, nil
}

func countSubs(subs []*pbgh.GithubIssue) (float64, float64) {
	total := float64(0)
	complete := float64(0)
	for _, sub := range subs {
		if sub.GetState() == pbgh.IssueState_ISSUE_STATE_CLOSED {
			complete++
		}
		total++

		nt, nc := countSubs(sub.GetSubIssues())
		total += nt
		complete += nc
	}

	return total, complete
}

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

	launch, err := getLaunchBug(ctx, "brotherlogic", "gramophile", 1760)
	if err != nil {
		return err
	}

	total := float64(0)
	complete := float64(0)

	if launch.GetState() == pbgh.IssueState_ISSUE_STATE_CLOSED {
		complete++
	}
	total++
	nt, nc := countSubs(launch.GetSubIssues())
	total += nt
	complete += nc

	launchTotal.With(prometheus.Labels{"version": "v1"}).Set(total)
	launchDone.With(prometheus.Labels{"version": "v1"}).Set(complete)

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

	oldestSale.Set(float64(stats.GetSaleStats().GetOldestLastUpdate()))
	oldestSaleId.Set(float64(stats.SaleStats.GetOldestId()))
	erdMissing.Set(float64(stats.GetCollectionStats().GetErdMissingCount()))

	wants.Set(float64(stats.GetCollectionStats().GetTotalWants()))
	syncedWants.Set(float64(stats.GetCollectionStats().GetSyncedWants()))

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
