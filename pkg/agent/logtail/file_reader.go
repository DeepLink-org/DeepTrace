// Copyright (c) OpenMMLab. All rights reserved.

package logtail

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"deeptrace/logger"
	"deeptrace/pkg/agent/util/scripts"
	"deeptrace/pkg/agent/util/textparser"
	pb "deeptrace/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func NewFileReader(ctx context.Context, req *pb.GetRecentLogsRequest) Interface {
	workDir := os.Getenv("WORK_DIR")
	if req != nil && req.WorkDir != "" {
		workDir = req.WorkDir
	}
	return &FileReader{
		workDir:   workDir,
		logParser: &textparser.LogParser{},
	}
}

// Get latest logs for all ranks
func (s *FileReader) GetRecentLogs(ctx context.Context, maxLines int32) ([]*pb.RankLog, error) {
	// 1. Get the latest log directory
	logDir, err := getLatestLogDir(s.workDir)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "Log directory not found: %v", err)
	}

	// 2. Determine the number of ranks (via environment variables)
	rankMin, rankMax, err := scripts.GetCurrentNodeRankRange(ctx)
	if err != nil {
		logger.Logger.Error("Failed to get rank range", zap.Error(err))
		return nil, err
	}
	rankRange := rankMax - rankMin + 1

	// 3. Collect all rank logs
	rankLogs := make([]*pb.RankLog, 0, rankRange)

	for rank := rankMin; rank < rankMax; rank++ {
		lines, fmodTime, err := readRankLogTail(logDir, rank, int(maxLines))
		if err != nil {
			// Partial failure doesn't affect other ranks
			lines = []string{fmt.Sprintf("Log reading failed: %v", err)}
		}

		// entries, latestTime := parceLogs(ctx, lines)
		entries, err := textparser.ParseWithType(ctx, s.logParser, lines)
		if err != nil {
			logger.Logger.Error("LogParser ParseWithType", zap.Error(err))
		}

		now := time.Now()
		ranklog := &pb.RankLog{
			Rank:     fmt.Sprintf("RANK%d", rank),
			TailTime: timestamppb.New(now),
		}
		if len(entries) > 0 {
			latestTime := getLatestTime(entries, fmodTime)
			// Default to 1970 or init value, set -10 if no valid time
			if latestTime.Before(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local)) {
				ranklog.SuspendSeconds = -10
			} else {
				ranklog.SuspendSeconds = int32(now.Sub(latestTime).Seconds())
			}

			ranklog.Entries = entries
		}

		rankLogs = append(rankLogs, ranklog)
	}

	return rankLogs, nil
}

func getLatestTime(entries []*pb.LogEntry, fmodTime time.Time) time.Time {
	latestTime := time.Date(1900, time.January, 1, 0, 0, 0, 0, time.Local)
	for _, entry := range entries {
		if entry.Timestamp.AsTime().After(latestTime) {
			latestTime = entry.Timestamp.AsTime()
		}
	}
	// Prefer log timestamps - fallback to file modification time if parsing fails
	if latestTime.Before(time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local)) {
		return fmodTime
	}
	return latestTime
}

// Get latest log directory (only one task at a time)
func getLatestLogDir(workDir string) (string, error) {
	files, err := os.ReadDir(workDir)
	if err != nil {
		return "", err
	}

	// Filter and sort timestamp directories
	var dirs []os.FileInfo
	for _, f := range files {
		if f.IsDir() {
			info, err := f.Info()
			if err != nil {

			}
			dirs = append(dirs, info)
		}
	}

	if len(dirs) == 0 {
		return "", fmt.Errorf("no log directories found")
	}

	// Sort by modification time (newest first)
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].ModTime().After(dirs[j].ModTime())
	})

	return filepath.Join(workDir, dirs[0].Name()), nil
}

// Read tail of specific rank's log
func readRankLogTail(logDir string, rank int, lines int) ([]string, time.Time, error) {
	var fileModTime time.Time
	logFile := filepath.Join(logDir, fmt.Sprintf("rank%d.log", rank))
	file, err := os.Open(logFile)
	if err != nil {
		return nil, fileModTime, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// Set 1MB buffer for long lines
	const maxCapacity = 1024 * 1024
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	var logLines []string
	lineCount := 0

	// Use ring buffer to store last N lines
	ringBuffer := make([]string, lines)
	index := 0

	for scanner.Scan() {
		ringBuffer[index] = scanner.Text()
		index = (index + 1) % lines
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		logger.Logger.Error("Log scanning error", zap.Error(err))
		return nil, fileModTime, fmt.Errorf("Log scanning error: %v", err)
	}

	// Populate actual results
	start := 0
	if lineCount > lines {
		start = index
	}

	for i := 0; i < min(lineCount, lines); i++ {
		pos := (start + i) % lines
		logLines = append(logLines, ringBuffer[pos])
	}

	if fstat, err := file.Stat(); err == nil {
		fileModTime = fstat.ModTime()
	} else {
		logger.Logger.Error("failed to fetch file stat", zap.Any("file", file.Name()), zap.Error(err))
	}

	return logLines, fileModTime, nil
}
