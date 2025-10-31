// Copyright (c) OpenMMLab. All rights reserved.

package checkhang

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"deeptrace/pkg/client/logs"
	"deeptrace/pkg/client/utils"
	"deeptrace/pkg/rules"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

type Result struct {
	node string
	logs *pb.LogResponse
	err  error
}

type CustomStackResult struct {
	ProcessType pb.ProcessType    `json:"processType"`
	Processes   []*pb.ProcessInfo `json:"processes"`
}

// Create checkhang command (intelligent hang detection)
func NewCmdCheckHang() *cobra.Command {
	var workDir string
	var maxLines int32
	var threshold int32

	cmd := &cobra.Command{
		Use:   "check-hang",
		Short: "Intelligent hang detection",
		Long: `Intelligently detect if the specified job is in a hang state.
Usage:
  client check-hang --job-id <job name> -w clusterx [--work-dir <working directory>] [--max-line <maximum lines>] [--threshold <preliminary judgment threshold for hang time>] [--interval-hang <automatic execution interval in minutes>] [--port <server port>]

Examples:
  client check-hang --job-id my_job -w clusterx --threshold 100 --interval-hang 5 --port 50051`,
		Run: func(cmd *cobra.Command, args []string) {
			jobName, _ := cmd.Flags().GetString("job-id")
			if jobName == "" {
				jobName = viper.GetString("job-id")
			}
			if jobName != "" {
				fmt.Printf("Using job name: %s\n", jobName)
			} else {
				fmt.Println("Note: No job name specified")
			}
			// Get worker source
			workSource, _ := cmd.Flags().GetString("worker-source")
			if workSource == "" {
				workSource = viper.GetString("worker-source")
			}
			if workSource == "" {
				fmt.Println("Error: worker source must be specified")
				os.Exit(1)
			}

			// Read address list
			addressList, err := utils.GetWorkerList(workSource, jobName)
			if err != nil {
				fmt.Printf("Failed to read address list file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Obtained addresses: %v\n", addressList)

			port, _ := cmd.Flags().GetString("port")
			if port == "" {
				port = viper.GetString("port")
				if port != "" {
					fmt.Printf("Using port number specified in configuration file: %s\n", port)
				} else {
					fmt.Println("No port number specified, using default value 50051")
					port = "50051"
				}
			} else {
				fmt.Printf("Using port number specified on command line: %s\n", port)
			}
			// If not specified by user, use configuration file or default value
			if workDir == "" {
				workDir = viper.GetString("work-dir")
				if workDir != "" {
					fmt.Printf("Using working directory specified in configuration file: %s\n", workDir)
				} else {
					fmt.Println("Note: No working directory specified, agent will try to read WORK_DIR environment variable")
				}
			} else {
				fmt.Printf("Using working directory specified on command line: %s\n", workDir)
			}
			if maxLines == 0 {
				maxLines = int32(viper.GetInt("max-line"))
				if maxLines != 0 {
					fmt.Printf("Using maximum log lines specified in configuration file: %d\n", maxLines)
				} else {
					fmt.Println("No maximum log lines specified, using default value 30")
					maxLines = 30
				}
			} else {
				fmt.Printf("Using maximum log lines specified on command line: %d\n", maxLines)
			}
			if threshold == 0 {
				threshold = int32(viper.GetInt("threshold"))
				if threshold != 0 {
					fmt.Printf("Using threshold specified in configuration file: %d\n", threshold)
				} else {
					fmt.Println("No threshold specified, using default value 120")
					threshold = 120
				}
			} else {
				fmt.Printf("Using threshold specified on command line: %d\n", threshold)
			}
			pollInterval, _ := cmd.Flags().GetInt("interval-hang")
			if pollInterval < 0 {
				fmt.Println("Please enter an appropriate time interval (minutes)")
				return
			}
			if pollInterval == 0 {
				pollInterval = viper.GetInt("interval-hang")
				if pollInterval == 0 {
					fmt.Println("Using default time interval of 1 minute")
					pollInterval = 1
				} else {
					fmt.Printf("Using time interval specified in configuration file: %d minutes\n", pollInterval)
				}
			} else {
				fmt.Printf("Using time interval specified on command line: %d minutes\n", pollInterval)
			}

			// Convert minutes to time.Duration type
			pollDuration := time.Duration(pollInterval) * time.Minute

			if pollInterval == 0 {
				fmt.Println("Execute detection only once, no polling")
				runCheckHang(jobName, workDir, maxLines, threshold, addressList, port)
				return
			}

			fmt.Printf("Starting intelligent detection, will automatically execute every %v...\n", pollDuration)
			fmt.Println("Press Ctrl+C to stop detection")

			// Execute first detection immediately
			runCheckHang(jobName, workDir, maxLines, threshold, addressList, port)

			// Use ticker to implement timed polling
			ticker := time.NewTicker(pollDuration)
			defer ticker.Stop()

			for range ticker.C {
				runCheckHang(jobName, workDir, maxLines, threshold, addressList, port)
			}
		},
	}

	// Add command line flags
	cmd.Flags().StringVar(&workDir, "work-dir", "", "Specify working directory")
	cmd.Flags().Int32Var(&maxLines, "max-line", 0, "Specify maximum log lines")
	cmd.Flags().Int32Var(&threshold, "threshold", 0, "Specify preliminary judgment threshold for hang time")
	cmd.Flags().IntP("interval-hang", "i", 0, "Automatic execution interval (minutes), 0 means execute only once")

	return cmd
}

func runCheckHang(jobName, workDir string, maxLines, threshold int32, addressList []string, port string) {
	if len(addressList) == 0 {
		os.Exit(1)
	}

	errorNodes := CheckLogs(jobName, addressList, workDir, maxLines, threshold, port)
	if len(errorNodes) > 0 {
		fmt.Printf("Nodes with errors: %v\n", errorNodes)
	}
}

func CheckLogs(job string, addressList []string, workDir string, maxLines int32, threshold int32, port string) []string {
	// Used to store nodes with suspendSeconds exceeding threshold
	suspiciousNodes := make(map[string]struct{})
	var mu sync.Mutex
	var wg sync.WaitGroup
	results := make(chan Result, len(addressList))
	// Start goroutines to process each node in parallel
	for _, node := range addressList {
		wg.Add(1)
		go func(node string) {
			defer wg.Done()

			conn, err := grpc.Dial(
				node+":"+port,
				grpc.WithInsecure(),
				grpc.WithTimeout(5*time.Second),
			)
			if err != nil {
				results <- Result{node, nil, err}
				fmt.Printf("Failed to connect to node %s: %v\n", node, err)
				return
			}
			defer conn.Close()

			client := pb.NewDeepTraceServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &pb.GetRecentLogsRequest{
				MaxLines: maxLines,
				WorkDir:  workDir,
			}

			resp, err := client.GetRecentLogs(ctx, req)
			if err != nil {
				// fmt.Printf("Failed to get logs from node %s: %v\n", node, err)
				results <- Result{node, nil, err}
				return
			}
			results <- Result{node, resp, err}
			// Check suspendSeconds for each rank
			for _, rankLog := range resp.Ranklogs {
				// Clean invalid UTF-8 strings
				for _, entry := range rankLog.Entries {
					entry.Message = utils.CleanUTF8(entry.Message)
				}

				// If suspendSeconds exceeds threshold, record the node
				if rankLog.SuspendSeconds > threshold {
					mu.Lock()
					suspiciousNodes[node] = struct{}{}
					mu.Unlock()
					fmt.Printf("Suspicious node %s found: suspendSeconds is %d (exceeds threshold %d)\n",
						node, rankLog.SuspendSeconds, threshold)
					CheckHangStacks(node, rankLog.Rank, port)
					break
				}
			}

		}(node)
	}

	// Wait for all goroutines to complete
	wg.Wait()
	date1 := time.Now()
	formattedTime := date1.Format("2006-01-02_15-04-05")
	finalResponse := &pb.LogResponse{}

	for i := 0; i < len(addressList); i++ {
		res := <-results
		if res.err != nil {
			fmt.Printf("Failed to get logs from node %s: %v\n", res.node, res.err)
			continue
		}
		// If rank is empty, add all logs
		finalResponse.Ranklogs = append(finalResponse.Ranklogs, res.logs.Ranklogs...)
	}
	close(results)
	customResponse := logs.CustomLogResponse{}
	for _, rankLog := range finalResponse.Ranklogs {
		var customEntries []logs.CustomLogEntry
		for _, entry := range rankLog.Entries {
			customEntries = append(customEntries, logs.CustomLogEntry{
				Timestamp: utils.FormatTimestamp(entry.Timestamp),
				Level:     entry.Level,
				Epoch:     entry.Epoch,
				Message:   entry.Message,
			})
		}
		customResponse.Ranklogs = append(customResponse.Ranklogs, logs.CustomRankLog{
			Rank:           rankLog.Rank,
			Entries:        customEntries,
			SuspendSeconds: rankLog.SuspendSeconds,
			TailTime:       utils.FormatTimestamp(rankLog.TailTime),
		})
	}
	jsonData, err := json.MarshalIndent(customResponse, "", "  ")
	if customResponse.Ranklogs != nil {
		// Convert custom structure results to JSON and output

		if err != nil {
			fmt.Printf("Failed to convert to JSON: %v\n", err)
		}
		fileName := fmt.Sprintf("%s_checkLogs_%s.json", job, formattedTime)
		err = utils.AppendWithTimestamp("checkLogs", fileName, jsonData)
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("Log information successfully saved to %s\n", fileName)
		}
	}
	// Convert map to slice
	result := make([]string, 0, len(suspiciousNodes))
	for node := range suspiciousNodes {
		result = append(result, node)
	}

	return result
}

// Save process information to file
func saveProcessesToFile(processes []*pb.ProcessInfo, filePath string, dir string) error {

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("Failed to create logs directory: %w", err)
	}

	// 3. Concatenate full file path (put filename in logs directory)
	fullPath := filepath.Join(dir, filePath)
	// Open file (create if not exists, append if exists)
	file, err := os.OpenFile(fullPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("Cannot open file: %v", err)
	}
	defer file.Close()

	// Convert structure to JSON format
	jsonMarshal := protojson.MarshalOptions{
		UseEnumNumbers: false,
		Multiline:      true,
		Indent:         " ",
	}
	data, err := jsonMarshal.Marshal(&pb.ProcessInfoList{
		Processes:  processes,
		TotalCount: int32(len(processes)),
	})
	if err != nil {
		return fmt.Errorf("Cannot convert process data to JSON: %v", err)
	}

	// Write data to file
	if _, err := file.Write(data); err != nil {
		return fmt.Errorf("Failed to write to file: %v", err)
	}

	// Add newline to ensure next append starts on a new line
	if _, err := file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("Failed to write newline: %v", err)
	}

	if _, err := file.Write([]byte("\n")); err != nil {
		return fmt.Errorf("Failed to write newline: %v", err)
	}
	return nil
}

func CheckHangStacks(node string, rank string, port string) {
	currentProcesses := make([][]*pb.ProcessInfo, 5)
	datee := make([]time.Time, 5)

	for i := 0; i < 5; i++ {
		conn, err := grpc.Dial(
			node+":"+port,
			grpc.WithInsecure(),
			grpc.WithTimeout(5*time.Second), // Increase connection timeout
		)
		if err != nil {
			fmt.Printf("Failed to connect to node %s: %v\n", node, err)
			continue
		}
		defer conn.Close()

		client := pb.NewDeepTraceServiceClient(conn)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		req := &pb.GetProcessStacksRequest{}

		resp, err := client.GetProcessStacks(ctx, req)
		if err != nil {
			fmt.Printf("Failed to get type stack information from node %s: %v\n", node, err)
			continue
		}

		customResult := CustomStackResult{
			ProcessType: 0,
			Processes:   resp.Processes,
		}

		if currentProcesses[i] == nil {
			currentProcesses[i] = make([]*pb.ProcessInfo, 0)
		}
		currentProcesses[i] = append(currentProcesses[i], customResult.Processes...)

		ctx = context.Background()
		if i == 0 {
			datee[i] = time.Now()
			formattedTime := datee[i].Format("2006-01-02_15-04-05")
			fileName := fmt.Sprintf("node%s_processInfo_%s.json", node, formattedTime)
			err1 := saveProcessesToFile(customResult.Processes, fileName, "checkStacks")
			if err1 != nil {
				fmt.Println("Error:", err1)
			} else {
				fmt.Printf("Process data successfully saved to %s\n", fileName)
			}
		} else {
			datee[i] = time.Now()
			fmt.Println("Start comparing", node, "at", datee[i].Format("2006-01-02 15:04:05"), "with", datee[i-1].Format("2006-01-02 15:04:05"), "training process stack information.")
			fmt.Println()
			b, oo, error := rules.PstreeEqual(ctx, datee[i].Format("2006-01-02 15:04:05"), datee[i-1].Format("2006-01-02 15:04:05"), currentProcesses[i], currentProcesses[i-1])
			if error != nil {
				fmt.Println(error)
			}
			fmt.Println("Detection results:")
			if !b {
				for _, diffline := range oo {
					fmt.Println(diffline.Diff)
					fmt.Println("--------------------------------------------------")
				}
				// If there's an issue, save JSON file
				formattedTime := datee[i].Format("2006-01-02_15-04-05")
				fileName := fmt.Sprintf("node%s_processInfo_%s_haveDiff.json", node, formattedTime)
				err1 := saveProcessesToFile(customResult.Processes, fileName, "checkStacks")
				if err1 != nil {
					fmt.Println("Error:", err1)
				} else {
					fmt.Printf("Process data successfully saved to %s\n", fileName)
				}
			} else {
				fmt.Println("No anomalies detected")
				formattedTime := datee[i].Format("2006-01-02_15-04-05")
				fileName := fmt.Sprintf("node%s_processInfo_%s_noDiff.json", node, formattedTime)
				err1 := saveProcessesToFile(customResult.Processes, fileName, "checkStacks")
				if err1 != nil {
					fmt.Println("Error:", err1)
				} else {
					fmt.Printf("Process data successfully saved to %s\n", fileName)
				}

			}
			fmt.Println("--------------------------------------------------")
		}

		// Sleep for 5 seconds
		time.Sleep(5 * time.Second)
	}
}
