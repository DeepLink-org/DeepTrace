// Copyright (c) OpenMMLab. All rights reserved.

package rules

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"deeptrace/logger"
	pb "deeptrace/v1"
)

// Comparison of process stack information for all training processes on a node at different times
func PstreeEqual(ctx context.Context, time1, time2 string, psa, psb []*pb.ProcessInfo) (bool, []ProccessInfoDiff, error) {
	if len(psa) != len(psb) {
		return false, nil, fmt.Errorf("process count mismatch: psa: %d, psb: %d", len(psa), len(psb))
	}
	sort.Slice(psa, func(i, j int) bool { return psa[i].Pid < psa[j].Pid })
	sort.Slice(psb, func(i, j int) bool { return psb[i].Pid < psb[j].Pid })
	equal := true
	diff := []ProccessInfoDiff{}

	for i := 0; i < len(psa); i++ {

		eq, dif, err := ThreadsStacksEqual(ctx, time1, time2, psa[i], psb[i])
		if err != nil {
			return false, nil, err
		}
		if !eq {
			equal = false
			diff = append(diff, ProccessInfoDiff{
				Rank:  psa[i].Rank,
				Diff:  dif,
				PType: psa[i].Type,
			})
		}
	}

	return equal, diff, nil
}

func ThreadsStacksEqual(ctx context.Context, time1, time2 string, proccessa, proccessb *pb.ProcessInfo) (equal bool, diff string, err error) {
	if proccessa == nil || proccessb == nil {
		logger.Logger.Error("process info is nil")
		return false, "", fmt.Errorf("process info is empty")
	}

	if !strings.EqualFold(proccessa.Rank, proccessb.Rank) {
		return false, "", fmt.Errorf("process ranks are different: %s and %s", proccessa.Rank, proccessb.Rank)
	}

	if proccessa.Pid != proccessb.Pid {
		return false, "", fmt.Errorf("internal error, comparing two different processes. pid: %d and %d", proccessa.Pid, proccessb.Pid)
	}

	if len(proccessa.Threads) != len(proccessb.Threads) {
		header := fmt.Sprintf("Detected different number of threads in process stack. %s, Process type: %s, Process ID:%d\n", proccessa.Rank, proccessa.Type.String(), proccessa.Pid)
		before := fmt.Sprintf("[%s]Thread count: %d\n", time1, len(proccessa.Threads))
		after := fmt.Sprintf("[%s]Thread count: %d\n", time2, len(proccessb.Threads))
		return false, header + before + after, nil
	}

	// threads sort
		sort.Slice(proccessa.Threads, func(i, j int) bool {
		return proccessa.Threads[i].ThreadId < proccessa.Threads[j].ThreadId
	})
	sort.Slice(proccessb.Threads, func(i, j int) bool {
		return proccessb.Threads[i].ThreadId < proccessb.Threads[j].ThreadId
	})

	for i := 0; i < len(proccessa.Threads); i++ {
		ta, tb := proccessa.Threads[i], proccessb.Threads[i]
		if ta.ThreadId != tb.ThreadId {
			header := fmt.Sprintf("Detected different thread IDs. %s, Process type: %s, Process ID:%d\n", proccessa.Rank, proccessa.Type.String(), proccessa.Pid)
			before := fmt.Sprintf("[%s]Thread ID: %d\n", time1, ta.ThreadId)
			after := fmt.Sprintf("[%s]Thread ID: %d\n", time2, tb.ThreadId)
			return false, header + before + after, nil
		}
		if len(ta.StackFrames) != len(tb.StackFrames) {
			header := fmt.Sprintf("Detected different number of thread stack frames. %s, Process type: %s, Process ID:%d, Thread ID: %d\n", proccessa.Rank, proccessa.Type.String(), proccessa.Pid, ta.ThreadId)
			before := fmt.Sprintf("[%s]Frame count: %d\n", time1, len(ta.StackFrames))
			after := fmt.Sprintf("[%s]Frame count: %d\n", time2, len(tb.StackFrames))
			return false, header + before + after, nil
		}

		var flevel int32
		for flevel = 0; flevel < int32(len(ta.StackFrames)); flevel++ {
			if !strings.EqualFold(ta.StackFrames[flevel], tb.StackFrames[flevel]) {
				header := fmt.Sprintf("Detected different process stacks. %s, Process type: %s, Process ID:%d, Thread ID: %d, Stack frame level: %d\n", proccessa.Rank, proccessa.Type.String(), proccessa.Pid, ta.ThreadId, flevel+1)
				before := fmt.Sprintf("[%s]Stack frame content: %s\n", time1, ta.StackFrames[flevel])
				after := fmt.Sprintf("[%s]Stack frame content: %s\n", time2, tb.StackFrames[flevel])

				return false, header + before + after, nil
			}
		}
	}

	return true, "", nil
}
