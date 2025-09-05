// Copyright (c) OpenMMLab. All rights reserved.

package textparser

import (
	"context"
	"regexp"
	"strconv"
	"time"

	pb "deeptrace/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// Log parser
type LogParser struct{}

func (p *LogParser) Parse(ctx context.Context, inputs []string) ([]*pb.LogEntry, error) {
	entries := make([]*pb.LogEntry, 0, len(inputs))
	for _, line := range inputs {
		entry := &pb.LogEntry{
			Message: line,
		}
		// Basic structure matching
		baseReg := regexp.MustCompile(`\[([^\]]+)\]\[RANK (\d+)\]\[([^\]]+)\]\[([^\]]+)\] (.*)`)
		baseMatches := baseReg.FindStringSubmatch(line)
		if len(baseMatches) < 6 {
			entries = append(entries, entry)
			continue
		}

		// Parse timestamp
		timestamp, err := time.Parse("2006-01-02 15:04:05", baseMatches[3])
		if err != nil {
			continue
		}

		entry.Timestamp = timestamppb.New(timestamp)
		entry.Level = pb.LogLevel(pb.LogLevel_value[baseMatches[4]])

		msgBody := baseMatches[5]
		epochReg := regexp.MustCompile(`\[Train\] \(Epoch (\d+)\)`)
		epochMatch := epochReg.FindStringSubmatch(msgBody)

		if epochMatch != nil {
			entry.Epoch = extractNumber(epochMatch[1], "(\\d+)")
		}

		entries = append(entries, entry)
	}

	return entries, nil
}

func extractNumber(s, pattern string) int32 {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(s)
	if len(match) > 1 {
		n, _ := strconv.Atoi(match[1])
		return int32(n)
	}
	return 0
}
