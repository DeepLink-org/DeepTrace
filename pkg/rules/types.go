// Copyright (c) OpenMMLab. All rights reserved.

package rules

import (
	pb "deeptrace/v1"
)

type ProccessInfoDiff struct {
	Rank  string
	Diff  string
	PType pb.ProcessType
}
