// Copyright (c) OpenMMLab. All rights reserved.

package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deeptrace/logger"
	"deeptrace/pkg/agent/grpcserver"
	"deeptrace/pkg/agent/httpserver"
	"deeptrace/pkg/agent/util/storage"
	"deeptrace/pkg/prom/metrics"
	"deeptrace/pkg/version"
	pb "deeptrace/v1"

	"github.com/gorilla/mux"
	"github.com/soheilhy/cmux"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

// 1. Define command-line arguments and environment variables
var (
	Port           = flag.String("port", "", "grpc service listen port, default 50051")
	PushGatewayURL = flag.String("push-gateway", "", "Pushgateway URL (e.g., http://localhost:9091)")
	JobName        = flag.String("job-name", "deeptraced", "Job name for metrics")
	PushInterval   = flag.Duration("push-interval", 15*time.Second, "Metrics push interval")
	PersistenceDir = flag.String("persistence-dir", "", "persistent directory for events such as trainning alert message. will use $WORK_DIR if unset. use /tmp if $WORK_DIR unset.")
)

func main() {
	flag.Parse()
	go metrics.PushMetricsToGateway(*PushGatewayURL, *JobName, *PushInterval)

	restartChan := make(chan struct{}, 1)
	storageC, err := storage.NewEventStorage(*PersistenceDir, 0)
	if err != nil {
		logger.Logger.Error("Failed to init storage", zap.Error(err))
		os.Exit(1)
	}
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(metrics.MetricsInterceptor),
	)
	pb.RegisterDeepTraceServiceServer(grpcServer, &grpcserver.TraceServiceServer{
		RestartChan: restartChan,
	})
	pb.RegisterAlertServiceServer(grpcServer, &grpcserver.AlertServiceServer{
		Storage: storageC,
	})

	// Get port from environment variable, default to 50051
	if *Port == "" {
		*Port = os.Getenv("DEEPTRACED_PORT")
	}
	if *Port == "" {
		*Port = "50051"
	}

	lis, err := net.Listen("tcp", ":"+(*Port))
	if err != nil {
		logger.Logger.Fatal("failed to listen: %v", zap.Error(err))
		os.Exit(1)
	}

	reflection.Register(grpcServer)

	// Create a multiplexer
	m := cmux.New(lis)

	// Match gRPC requests
	grpcL := m.MatchWithWriters(cmux.HTTP2MatchHeaderFieldSendSettings("content-type", "application/grpc"))

	// Match HTTP requests
	httpL := m.Match(cmux.HTTP1Fast())

	// Goroutine to listen for restart signals
	go func() {
		<-restartChan
		logger.Logger.Info("Received restart request. Gracefully stopping...")

		// Gracefully stop the gRPC service
		grpcServer.GracefulStop()
		logger.Logger.Info("gRPC server stopped")

		// Exit the process (restarted by the process manager)
		os.Exit(0)
	}()

	// Operating system signal handling
	go func() {
		stopChan := make(chan os.Signal, 1)
		signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)
		sig := <-stopChan
		logger.Logger.Info("Received system signal. Shutting down...", zap.Any("sig", sig))
		grpcServer.GracefulStop()

		os.Exit(0)
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	group, _ := errgroup.WithContext(ctx)

	logger.Logger.Info("Starting service", zap.Any("version", version.GetAgentVersionInfo()))
	group.Go(func() error {
		logger.Logger.Info("gRPC server listening at", zap.String("addr", grpcL.Addr().String()))
		return grpcServer.Serve(grpcL)
	})

	// Start HTTP service
	group.Go(func() error {
		// Create router
		router := mux.NewRouter()

		// Register HTTP handlers
		httpHandler := httpserver.NewDefaultHandler(storageC)
		httpHandler.RegisterRoutes(router)

		// Add health check
		router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})

		// Create HTTP server
		httpServer := &http.Server{
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
		}

		logger.Logger.Info("HTTP server listening at", zap.String("addr", httpL.Addr().String()))
		return httpServer.Serve(httpL)
	})

	// Start the multiplexer
	group.Go(func() error {
		return m.Serve()
	})

	// Wait for service to exit
	if err := group.Wait(); err != nil {
		logger.Logger.Error("Server error", zap.Error(err))
	}
}
