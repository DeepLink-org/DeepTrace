// Copyright (c) OpenMMLab. All rights reserved.

package logtail

import (
	"context"

	"deeptrace/pkg/agent/util/textparser"
	pb "deeptrace/v1"
)

type Interface interface {
	GetRecentLogs(ctx context.Context, maxLines int32) ([]*pb.RankLog, error)
}

type FileReader struct {
	workDir   string
	logParser *textparser.LogParser
}
