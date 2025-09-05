// Copyright (c) OpenMMLab. All rights reserved.

package metrics

import (
	"context"
	"net"
	"testing"

	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestMetricsInterceptor(t *testing.T) {
	// Create a simple gRPC service to test the interceptor
	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer(grpc.UnaryInterceptor(MetricsInterceptor))
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Errorf("Server exited with error: %v", err)
		}
	}()
	defer server.Stop()

	// Create client connection
	conn, err := grpc.DialContext(context.Background(), "bufnet", grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}), grpc.WithInsecure())
	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	defer conn.Close()

	// Specific gRPC calls can be added here to test the interceptor
	// Since we don't have an actual gRPC service, we can only test the basic functionality of the interceptor
	ctx := context.Background()
	req := "test request"
	info := &grpc.UnaryServerInfo{
		FullMethod: "/test.Service/TestMethod",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return "test response", nil
	}

	// Call the interceptor
	resp, err := MetricsInterceptor(ctx, req, info, handler)

	if err != nil {
		t.Errorf("MetricsInterceptor returned an error: %v", err)
	}

	if resp != "test response" {
		t.Errorf("MetricsInterceptor returned unexpected response: %v", resp)
	}
}
