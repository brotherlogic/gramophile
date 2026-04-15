package server

import (
	"log"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/metric"
)

var (
	meter                     metric.Meter
	grpcServerHandledTotal    metric.Int64Counter
	grpcServerHandlingSeconds metric.Float64Histogram
)

func initMetrics() {
	meter = otel.Meter("github.com/brotherlogic/gramophile/server")
	var err error
	grpcServerHandledTotal, err = meter.Int64Counter("grpc_server_handled_total",
		metric.WithDescription("Total number of RPCs completed on the server, regardless of success or failure."),
	)
	if err != nil {
		log.Printf("Failed to create counter: %v", err)
	}

	grpcServerHandlingSeconds, err = meter.Float64Histogram("grpc_server_handling_seconds",
		metric.WithDescription("The total time it took to handle the RPC on the server."),
		metric.WithUnit("s"),
	)
	if err != nil {
		log.Printf("Failed to create histogram: %v", err)
	}
}
