// Copyright (c) OpenMMLab. All rights reserved.

package metrics

import (
	"context"
	"deeptrace/logger"
	"os"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/push"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var (
	// Request counter
	RequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "grpc_requests_total",
		Help: "Total number of gRPC requests",
	}, []string{"method", "status"})

	// Request latency histogram
	RequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "grpc_request_duration_seconds",
		Help:    "Duration of gRPC requests in seconds",
		Buckets: prometheus.DefBuckets,
	}, []string{"method"})
)

// gRPC interceptor (used for automatic metric collection)
func MetricsInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	method := info.FullMethod

	resp, err := handler(ctx, req)

	duration := time.Since(start).Seconds()
	status := "success"
	if err != nil {
		status = "error"
	}

	RequestsTotal.WithLabelValues(method, status).Inc()
	RequestDuration.WithLabelValues(method).Observe(duration)

	return resp, err
}

func PushMetricsToGateway(pushgatewayUrl, jobName string, interval time.Duration) {
	if pushgatewayUrl == "" {
		logger.Logger.Error("Pushgateway URL not set, skipping metrics push")
		return
	}

	pusher := push.New(pushgatewayUrl, jobName).
		Collector(RequestsTotal).
		Collector(RequestDuration).
		Grouping("instance", getHostname())

	for {
		<-time.After(interval)
		if err := pusher.Push(); err != nil {
			logger.Logger.Error("Error pushing metrics", zap.Error(err))
		}
	}
}

func getHostname() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}

	if hostname := os.Getenv("HOSTNAME"); hostname != "" {
		return hostname
	}

	if hostname := os.Getenv("HOST"); hostname != "" {
		return hostname
	}

	if data, err := os.ReadFile("/etc/hostname"); err == nil {
		return string(data)
	}

	return "unknown"
}
