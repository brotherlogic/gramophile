package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"

	pb "github.com/brotherlogic/gramophile/proto"
	"github.com/brotherlogic/gramophile/server"
)

var (
	port         = flag.Int("port", 8080, "The server port for grpc traffic")
	metricsPort  = flag.Int("metrics_port", 8081, "Metrics port")
	httpPort     = flag.Int("http_port", 8082, "Port to serve regular http traffic")
	internalPort = flag.Int("internal_port", 8083, "Port to serve internal grpc traffic")
)

func main() {
	flag.Parse()

	s := server.NewServer()

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("gramophile is unable to listen on the grpc port %v: %v", *port, err)
	}
	gs := grpc.NewServer()
	pb.RegisterGramophileEServiceServer(gs, s)
	go func() {
		if err := gs.Serve(lis); err != nil {
			log.Fatalf("gramophile is unable to serve grpc: %v", err)
		}
		log.Printf("gramophile has closed the grpc port")
	}()

	lis, err = net.Listen("tcp", fmt.Sprintf(":%d", *internalPort))
	if err != nil {
		log.Fatalf("gramophile is unable to listen on the internal grpc port %v: %v", *internalPort, err)
	}
	gsInternal := grpc.NewServer()
	pb.RegisterGramophileServiceServer(gsInternal, s)
	pb.RegisterQueueServiceServer(gsInternal, s.Queue)
	go func() {
		if err := gsInternal.Serve(lis); err != nil {
			log.Fatalf("gramophile is unable to serve internal grpc: %v", err)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
		log.Fatalf("gramophile is unable to serve metrics: %v", err)
	}()

	mux := http.NewServeMux()
	mux.Handle("/callback", s)
	err = http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux)
	log.Fatalf("gramophile is unable to serve http: %v", err)
}
