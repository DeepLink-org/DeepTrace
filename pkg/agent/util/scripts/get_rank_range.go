// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"text/template"

	"deeptrace/logger"

	"go.uber.org/zap"
)

const rankRangeTemplate = `#!/bin/bash

# 1. Find launcher process
launcher_pid=$(ps aux | grep -m1 "torchrun\|distributed.launch" | grep -v grep | awk '{print $2}')

# Check if launcher process is found
if [[ -z "$launcher_pid" ]]; then
    echo "error: No launcher process found"
    exit 1
fi

#echo "[Launcher] PID: $launcher_pid"

# 2. Collect Ranks of all training processes
declare -A train_pids
ranks=()  # Store all valid Rank values
found_ranks=0

# Only check direct child processes of the launcher
for pid in $(pgrep -P $launcher_pid); do
    # Get process environment variables
    env_data=$(tr '\0' '\n' < /proc/$pid/environ 2>/dev/null)
    
    # Check if RANK variable exists
    rank=$(grep "^RANK=" <<< "$env_data" | cut -d= -f2)
    
    # Check if RANK is valid (numeric)
    if [[ -n "$rank" && "$rank" =~ ^[0-9]+$ ]]; then
        gpu_id=$(grep "^LOCAL_RANK=" <<< "$env_data" | cut -d= -f2)
        train_pids[$rank]="$pid:$gpu_id"
        #echo "[Train Process] RANK=$rank PID=$pid GPU=$gpu_id"
        
        # Add to Rank list
        ranks+=($rank)
        ((found_ranks++))
    fi
done

# 3. Check if valid Ranks are found
if [[ $found_ranks -eq 0 ]]; then
    echo "error: No valid training processes with RANK found"
    exit 1
fi

# 4. Calculate Rank range
# Sort Ranks
sorted_ranks=($(printf "%s\n" "${ranks[@]}" | sort -n))

min_rank=${sorted_ranks[0]}
max_rank=${sorted_ranks[${#sorted_ranks[@]}-1]}

# 5. Output Rank range
echo "$min_rank:$max_rank"
exit 0
`

// Get the rank range of the current node
func GetCurrentNodeRankRange(ctx context.Context) (minNum, maxNum int, err error) {
	tmpl, err := template.New("rankRange").Parse(rankRangeTemplate)
	if err != nil {
		return 0, 0, err
	}

	output, err := executeScript(ctx, tmpl)
	if err != nil {
		return 0, 0, err
	}
	logger.Logger.Info("GetCurrentNodeRankRange ", zap.String("output:", string(output)))

	// Parse output
	return parseRankRangeOutput(string(output))
}

// Parse script output
func parseRankRangeOutput(output string) (minNum, maxNum int, err error) {
	output = strings.TrimSpace(output)
	if strings.HasPrefix(output, "error:") {
		return 0, 0, fmt.Errorf("rank range script error: %s", output)
	}

	parts := strings.Split(output, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("Invalid output format: %s", output)
	}

	minRank, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("Failed to parse minRank: %v", err)
	}

	maxRank, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("Failed to parse maxRank: %v", err)
	}

	return minRank, maxRank, nil
}
