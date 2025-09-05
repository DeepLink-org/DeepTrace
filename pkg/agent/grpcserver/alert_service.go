// Copyright (c) OpenMMLab. All rights reserved.

package grpcserver

import (
	"context"
	"time"

	"deeptrace/logger"
	"deeptrace/pkg/agent/util/storage"
	pb "deeptrace/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AlertServiceServer struct {
	pb.UnimplementedAlertServiceServer
	Storage *storage.EventStorage
}

// GetAlerts retrieves alerts based on the request parameters.
//
// Parameters:
//   - ctx: The context for the request.
//   - req: The GetAlertsRequest containing parameters for alert retrieval.
//
// Returns:
//   - *pb.GetAlertsResponse: The response containing alert records.
//   - error: An error if alert retrieval fails.
func (s *AlertServiceServer) GetAlerts(ctx context.Context, req *pb.GetAlertsRequest) (*pb.GetAlertsResponse, error) {
	startTime, endTime := int64(0), time.Now().UnixMilli()

	if req.StartTime != nil {
		startTime = req.StartTime.AsTime().UnixMilli()
	}
	if req.EndTime != nil {
		endTime = req.EndTime.AsTime().UnixMilli()
	}

	alerts, err := s.Storage.LoadEvents(storage.EventFilter{
		StartTime:   startTime,
		EndTime:     endTime,
		MinSeverity: int32(req.MinSeverity),
		Type:        "alert",
		Unprocessed: req.Unprocessed,
	})
	if err != nil {
		logger.Logger.Error("Failed to load alerts", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to load alerts: %v", err)
	}

	resp := &pb.GetAlertsResponse{}
	for _, a := range alerts {
		resp.Alerts = append(resp.Alerts, &pb.AlertRecord{
			Message:   a.Message,
			Timestamp: timestamppb.New(time.UnixMilli(a.Timestamp)),
			Severity:  pb.Severity(a.Severity),
		})
	}
	return resp, nil
}
