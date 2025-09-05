// Copyright (c) OpenMMLab. All rights reserved.

package stacktrace

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"deeptrace/logger"
	"deeptrace/pkg/agent/util/scripts"
	"deeptrace/pkg/agent/util/textparser"
	pb "deeptrace/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ Interface = &PythonStack{}

func NewPythonStack(ctx context.Context, maxConcurrent int, req *pb.GetProcessStacksRequest) Interface {

	return &PythonStack{
		sem: make(chan struct{}, maxConcurrent),
		req:        req,
		textParser: &textparser.StackParser{},
	}
}

// Get process stacks by type
func (s *PythonStack) GetProcessStacks(ctx context.Context) ([]*pb.ProcessInfo, error) {
	processesInfo := make([]*pb.ProcessInfo, 0)
	processes, err := scripts.GetProcessInfo(ctx)
	if err != nil {
		logger.Logger.Error("Failed to get training process information", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "Failed to get training process information: %v", err)
	}

	// Iterate through each process
	wg := sync.WaitGroup{}
	mu := sync.Mutex{}
	errs := make([]error, len(processes))
	for _, proc := range processes {
		proTpe := processType(proc.Type)
		rank := fmt.Sprintf("RANK%d", proc.Rank)
		if s.req.ProcessType != 0 && proTpe != s.req.ProcessType {
			continue
		}
		if len(s.req.Rank) != 0 && !strings.EqualFold(s.req.Rank, rank) {
			continue
		}
		procInfo := &pb.ProcessInfo{
			Type:      proTpe,
			Pid:       int32(proc.PID),
			Ppid:      int32(proc.PPID),
			Rank:      rank,
			LocalRank: fmt.Sprintf("RANK%d", proc.LocalRank),
		}

		wg.Add(1)
		go func(proc *pb.ProcessInfo) {
			defer wg.Done()
			// Call Fetch with semaphore control
			stack, err := s.Fetch(int(proc.Pid))
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				errs = append(errs, err)
				return
			}

			if threadsStack, err := textparser.ParseWithType(ctx, s.textParser, []string{stack}); err == nil {
				procInfo.Threads = threadsStack
			}

			processesInfo = append(processesInfo, procInfo)
		}(procInfo)
	}

	wg.Wait()

	return processesInfo, errors.Join(errs...)
}

func (f *PythonStack) Fetch(pid int) (string, error) {
	f.sem <- struct{}{}
	defer func() { <-f.sem }()

	return pyStack(pid)
}

func processType(typ string) pb.ProcessType {
	switch typ {
	case "trainer":
		return pb.ProcessType_PROCESS_TRAINER
	case "dataloader":
		return pb.ProcessType_PROCESS_DATA_LOADER
	default:
		return pb.ProcessType_PROCESS_UNSPECIFIED
	}
}

// Get stack using pystack
func pyStack(pid int) (string, error) {
	cmd := exec.Command("pystack", "remote", strconv.Itoa(pid))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pystack error: %v\noutput: %s", err, output)
	}
	return string(output), nil
}
