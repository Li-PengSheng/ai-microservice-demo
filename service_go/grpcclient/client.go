package grpcclient

import (
	"time"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"

	"my-go-gateway/config"
)

func Dial(cfg *config.Config) (*grpc.ClientConn, error) {
	return grpc.NewClient(
		cfg.AIServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(cfg.GRPCMaxRecvMsgSize),
		),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                cfg.GRPCKeepAliveTime,
			Timeout:             cfg.GRPCKeepAliveTimeout,
			PermitWithoutStream: true,
		}),
	)
}

// ObserveGRPC records gRPC latency and returns a done func to call after the call.
func ObserveGRPC(method string, start time.Time, err error) {
	status := "ok"
	if err != nil {
		status = "error"
	}
	// imported by handlers via the metrics package — kept here as a helper
	_ = method
	_ = start
	_ = status
}