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
	port        = flag.Int("port", 8080, "The server port for grpc traffic")
	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")
	httpPort    = flag.Int("http_port", 8082, "Port to serve regular http traffic")
)

type Server struct {
}

func main() {
	flag.Parse()

	s := &server.Server{}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("gramophile is unable to listen on the grpc serving port %v: %v", *port, err)
	}
	gs := grpc.NewServer()
	pb.RegisterGramophileEServiceServer(gs, s)
	pb.RegisterGramophileServiceServer(gs, s)

	// Setup prometheus export
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		err := http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
		log.Fatalf("gramophile is unable to serve metrics: %v", err)
	}()

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/callback", s)
		err := http.ListenAndServe(fmt.Sprintf(":%d", *httpPort), mux)
		log.Fatalf("gramophile is unable to serve http: %v", err)
	}()

	if err := gs.Serve(lis); err != nil {
		log.Fatalf("gramophile is unable to serve grpc: %v", err)
	}
}
