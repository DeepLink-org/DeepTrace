// Copyright (c) OpenMMLab. All rights reserved.

package scripts

import (
	"context"
	"encoding/json"
	"fmt"
	"text/template"

	"deeptrace/logger"

	"go.uber.org/zap"
)

const processStackTemplate = `#!/bin/bash

# Get PIDs of all torchrun launcher processes
launcher_pids=$(pgrep -f torchrun)

# Check if launcher processes are found
if [ -z "$launcher_pids" ]; then
    echo '{"error": "No torchrun launcher processes found"}' >&2
    exit 1
fi

# Prepare output JSON array
echo '['
first_entry=true

# Iterate through each launcher process
for launcher_pid in $launcher_pids; do
    # Get direct child processes (training processes) of the launcher
    train_pids=$(pgrep -P $launcher_pid)
    
    # Iterate through each training process
    for train_pid in $train_pids; do
        # Get environment variable information of the training process
        rank=""
        local_rank=""
        
        # Get environment variables from /proc filesystem
        if [ -f /proc/$train_pid/environ ]; then
            # Parse RANK and LOCAL_RANK
            rank=$(cat /proc/$train_pid/environ 2>/dev/null | tr '\0' '\n' | grep "^RANK=" | cut -d= -f2)
            local_rank=$(cat /proc/$train_pid/environ 2>/dev/null | tr '\0' '\n' | grep "^LOCAL_RANK=" | cut -d= -f2)
        fi
        
        # Output training process if rank information is found
        if [ -n "$rank" ]; then
            # Add comma separator (if not the first element)
            if [ "$first_entry" = true ]; then
                first_entry=false
            else
                echo ','
            fi
            
            # Output training process JSON object
            echo -n "  {\"type\": \"trainer\", \"pid\": $train_pid, \"ppid\": $launcher_pid, \"rank\": $rank, \"local_rank\": ${local_rank:-\"\"}}"
            
            # Get direct child processes (DataLoader workers) of the training process
            worker_pids=$(pgrep -P $train_pid -x 'pt_data_worker')
            
            # Iterate through each DataLoader worker
            for worker_pid in $worker_pids; do
                # Add comma separator
                echo ','
                
                # Output DataLoader worker JSON object
                echo -n "  {\"type\": \"dataloader\", \"pid\": $worker_pid, \"ppid\": $train_pid, \"rank\": $rank, \"local_rank\": ${local_rank:-\"\"}}"
            done
        fi
    done
done

echo
echo ']'
`

// Get training-related process information for the current node
func GetProcessInfo(ctx context.Context) ([]ProcessInfo, error) {
	tmpl, err := template.New("rankRange").Parse(processStackTemplate)
	if err != nil {
		return nil, err
	}

	return getTrainingProcesses(ctx, tmpl)
}

func getTrainingProcesses(ctx context.Context, tmpl *template.Template) ([]ProcessInfo, error) {
	output, err := executeScript(ctx, tmpl)
	if err != nil {
		logger.Logger.Error("Failed to execute process information script",
			zap.Error(err),
			zap.String("output", string(output)))
		return nil, err
	}

	// Try to parse as error
	var jsonErr struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(output, &jsonErr); err == nil && jsonErr.Error != "" {
		return nil, fmt.Errorf("get processed error: %s", jsonErr.Error)
	}

	// Parse as process information array
	var processes []ProcessInfo
	if err := json.Unmarshal(output, &processes); err != nil {
		logger.Logger.Error("Failed to parse process information JSON",
			zap.Error(err),
			zap.String("output", string(output)))
		return nil, fmt.Errorf("Failed to parse JSON: %v", err)
	}

	return processes, nil
}
