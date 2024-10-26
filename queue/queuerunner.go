package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"google.golang.org/grpc"

	"github.com/brotherlogic/discogs"
	"github.com/brotherlogic/gramophile/background"
	"github.com/brotherlogic/gramophile/db"
	"github.com/brotherlogic/gramophile/queuelogic"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	pstore_client "github.com/brotherlogic/pstore/client"

	pb "github.com/brotherlogic/gramophile/proto"
)

var (
	internalPort = flag.Int("internal_port", 8080, "GRPC port")
	metricsPort  = flag.Int("metrics_port", 8081, "Metrics port")
)

func main() {
	pstorec, err := pstore_client.GetClient()
	if err != nil {
		log.Fatalf("unable to connect to pstore: %v", err)
	}
	db := db.NewDatabase(context.Background())
	queue := queuelogic.GetQueue(
		pstorec,
		background.GetBackgroundRunner(db, os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK")),
		discogs.DiscogsWithAuth(os.Getenv("DISCOGS_KEY"), os.Getenv("DISCOGS_SECRET"), os.Getenv("DISCOGS_CALLBACK")), db)
	lis, err2 := net.Listen("tcp", fmt.Sprintf(":%d", *internalPort))
	if err2 != nil {
		log.Fatalf("gramophile is unable to listen on the internal grpc port %v: %v", *internalPort, err)
	}
	gsInternal := grpc.NewServer()
	pb.RegisterQueueServiceServer(gsInternal, queue)
	go func() {
		if err := gsInternal.Serve(lis); err != nil {
			log.Fatalf("queue is unable to serve internal grpc: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
		log.Fatalf("gramophile is unable to serve metrics -> %v", err)
	}()

	queue.Run()
}
