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
	port        = flag.Int("port", 8080, "The server port.")
	metricsPort = flag.Int("metrics_port", 8081, "Metrics port")
)

type Server struct {
}

func main() {
	flag.Parse()

	s := &server.Server{}

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("gramophile failed to listen on the main serving port %v: %v", *port, err)
	}
	gs := grpc.NewServer()
	pb.RegisterGramophileEServiceServer(gs, s)

	log.Printf("gramophile is listening on %v", lis.Addr())

	// Setup prometheus export
	http.Handle("/metrics", promhttp.Handler())
	go func() {
		http.ListenAndServe(fmt.Sprintf(":%v", *metricsPort), nil)
	}()

	if err := gs.Serve(lis); err != nil {
		log.Fatalf("gramophile failed to serve: %v", err)
	}
}
