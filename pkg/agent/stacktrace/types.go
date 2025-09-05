// Copyright (c) OpenMMLab. All rights reserved.

package stacktrace

import (
	"context"

	"deeptrace/pkg/agent/util/textparser"
	pb "deeptrace/v1"
)

type Interface interface {
	GetProcessStacks(ctx context.Context) ([]*pb.ProcessInfo, error)
}

type PythonStack struct {
	sem        chan struct{}
	req        *pb.GetProcessStacksRequest
	textParser *textparser.StackParser
}
