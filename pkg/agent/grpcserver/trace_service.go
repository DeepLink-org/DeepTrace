// Copyright (c) OpenMMLab. All rights reserved.

package grpcserver

import (
	"context"

	"deeptrace/logger"
	"deeptrace/pkg/agent/logtail"
	"deeptrace/pkg/agent/stacktrace"
	"deeptrace/pkg/version"
	pb "deeptrace/v1"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type TraceServiceServer struct {
	pb.UnimplementedDeepTraceServiceServer
	RestartChan chan struct{}
}

// GetRecentLogs retrieves recent logs based on the request parameters.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The GetRecentLogsRequest containing parameters for log retrieval.
//
// Returns:
//   - *pb.LogResponse: The response containing recent logs.
//   - error: An error if log retrieval fails.
func (s *TraceServiceServer) GetRecentLogs(ctx context.Context, req *pb.GetRecentLogsRequest) (*pb.LogResponse, error) {
	logtailclient := logtail.NewFileReader(ctx, req)
	rankLogs, err := logtailclient.GetRecentLogs(ctx, req.MaxLines)
	if err != nil {
		logger.Logger.Error("GetRecentLogs failed", zap.Error(err))
		return nil, err
	}
	return &pb.LogResponse{
		Ranklogs: rankLogs,
	}, nil
}

// GetProcessStacks retrieves process stacks based on the request parameters.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The GetProcessStacksRequest containing parameters for stack retrieval.
//
// Returns:
//   - *pb.ProcessStacksResponse: The response containing process stacks.
//   - error: An error if stack retrieval fails.
func (s *TraceServiceServer) GetProcessStacks(ctx context.Context, req *pb.GetProcessStacksRequest) (*pb.ProcessStacksResponse, error) {
	stackTraceClient := stacktrace.NewPythonStack(ctx, 72, req)
	proccesses, err := stackTraceClient.GetProcessStacks(ctx)
	if err != nil {
		logger.Logger.Error("GetProcessStacks failed", zap.Error(err))
		return nil, err
	}
	return &pb.ProcessStacksResponse{
		TotalProcesses: int32(len(proccesses)),
		Processes:      proccesses,
		SnapshotTime:   timestamppb.Now().String(),
	}, nil
}

// RestartServer restarts the server based on the request parameters.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The RestartRequest containing parameters for server restart.
//
// Returns:
//   - *pb.RestartResponse: The response indicating restart status.
//   - error: An error if restart fails.
func (s *TraceServiceServer) RestartServer(ctx context.Context, req *pb.RestartRequest) (*pb.RestartResponse, error) {
	// Trigger restart (non-blocking)
	select {
	case s.RestartChan <- struct{}{}:
	default: // Avoid blocking
	}

	return &pb.RestartResponse{
		Success: true,
		Message: "Restart initiated",
	}, nil
}

// GetVersion retrieves the version information.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: An empty request.
//
// Returns:
//   - *pb.VersionResponse: The response containing version information.
//   - error: An error if version retrieval fails.
func (s *TraceServiceServer) GetVersion(ctx context.Context, req *emptypb.Empty) (*pb.VersionResponse, error) {
	vi := version.GetStructuredVersion()
	return &pb.VersionResponse{
		Version:   vi.AgentVersion,
		Commit:    vi.Commit,
		BuildTime: vi.BuildTime,
		BuildTag:  vi.BuildTag,
	}, nil
}
