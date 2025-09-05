// Copyright (c) OpenMMLab. All rights reserved.

package scripts

type ProcessInfo struct {
	Type      string `json:"type"` // "trainer" / "dataloader"
	PID       int    `json:"pid"`
	PPID      int    `json:"ppid"`
	Rank      int    `json:"rank"`
	LocalRank int    `json:"local_rank"`
}
