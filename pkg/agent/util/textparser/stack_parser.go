// Copyright (c) OpenMMLab. All rights reserved.

package textparser

import (
	"bufio"
	"context"
	"regexp"
	"strconv"
	"strings"

	"deeptrace/logger"
	pb "deeptrace/v1"

	"go.uber.org/zap"
)

type StackParser struct{}

func (p *StackParser) Parse(ctx context.Context, inputs []string) ([]*pb.ThreadStack, error) {
	threads := []*pb.ThreadStack{}
	for _, input := range inputs {
		current := (*pb.ThreadStack)(nil)
		scanner := bufio.NewScanner(strings.NewReader(input))
		for scanner.Scan() {
			line := scanner.Text()
			switch {
			case isThreadStart(line):
				if current != nil {
					threads = append(threads, current)
				}
				current = parseThreadHeader(line)

			case isPythonFrame(line):
				if current != nil {
					frame := cleanFrame(line)
					current.StackFrames = append(current.StackFrames, frame)
				}

			case isCodeLine(line):
				if current != nil && len(current.StackFrames) > 0 {
					current.StackFrames[len(current.StackFrames)-1] += "\n" + strings.TrimSpace(line)
				}
			}
		}

		if current != nil {
			threads = append(threads, current)
		}
	}

	return threads, nil
}

func isThreadStart(line string) bool {
	return strings.HasPrefix(line, "Traceback for thread")
}

func isPythonFrame(line string) bool {
	return strings.Contains(line, "(Python) File")
}

func isCodeLine(line string) bool {
	return strings.HasPrefix(line, "    ") &&
		!strings.Contains(line, "File") &&
		strings.TrimSpace(line) != ""
}

func cleanFrame(line string) string {
	cleaned := strings.Replace(line, "(Python) ", "", 1)
	return strings.TrimSpace(cleaned)
}

func parseThreadHeader(line string) *pb.ThreadStack {
	re := regexp.MustCompile(`Traceback for thread (\d+) \((.*?)\)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) < 3 {
		return &pb.ThreadStack{
			ThreadId:   -1,
			ThreadName: "unknown",
		}
	}

	tid, err := strconv.Atoi(matches[1])
	if err != nil {
		logger.Logger.Error("Failed to conv threadId", zap.String("threadId", matches[1]), zap.Error(err))
		tid = -1
	}

	return &pb.ThreadStack{
		ThreadId:    int32(tid),
		ThreadName:  matches[2],
		StackFrames: []string{},
	}
}
