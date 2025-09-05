// Copyright (c) OpenMMLab. All rights reserved.

package logs

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"deeptrace/pkg/client/utils"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

// Create a customizable structure to store the converted results
type CustomLogEntry struct {
	Timestamp string      `json:"timestamp"`
	Level     pb.LogLevel `json:"level,omitempty"`
	Epoch     int32       `json:"epoch,omitempty"`
	Message   string
}

type CustomRankLog struct {
	Rank           string           `json:"rank"`
	Entries        []CustomLogEntry `json:"entries"`
	SuspendSeconds int32            `json:"suspend_seconds"`
	TailTime       string           `json:"tail_time"`
}

type CustomLogResponse struct {
	Ranklogs []CustomRankLog `json:"ranklogs"`
}

type Result struct {
	node string
	logs *pb.LogResponse
	err  error
}

// NewCmdLogs creates a cobra command for fetching logs via gRPC
func NewCmdLogs() *cobra.Command {
	var workDir string
	var maxLines int32

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Get log information",
		Long: `Get log information for the specified job.
Usage:
  client logs --job-id <job name> -a address_file [--work-dir <working directory>] [--max-line <maximum lines>] [--port <server port>]

Examples:
  client logs --job-id my_job -a address_file --work-dir /mnt/shared-storage --max-line 30 --port 50052`,
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

			// Get address list file path
			addressListPath, _ := cmd.Flags().GetString("address-list")
			if addressListPath == "" {
				addressListPath = viper.GetString("address-list")
			}
			if addressListPath == "" {
				fmt.Println("Error: Must specify the path to the agent address list file")
				os.Exit(1)
			}
			// Read address list
			addressList, err := utils.ReadAddressListFromFile(addressListPath)
			if err != nil {
				fmt.Printf("Failed to read address list file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Obtained addresses: %v\n", addressList)

			rank, _ := cmd.Flags().GetString("rank")
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

			FetchRankLogs(jobName, addressList, workDir, maxLines, rank, port)
		},
	}

	// Add --work-dir and --max-line flags
	cmd.Flags().StringVar(&workDir, "work-dir", "", "Specify working directory")
	cmd.Flags().Int32Var(&maxLines, "max-line", 0, "Specify maximum log lines")
	cmd.Flags().String("rank", "", "rank number, if not specified, return all ranks")
	_ = cmd.Flags().MarkHidden("rank")

	return cmd
}

func FetchRankLogs(jobName string, addressList []string, workDir string, maxLines int32, rank string, port string) {
	results := make(chan Result, len(addressList))

	// Start goroutines to process each node in parallel
	for _, addr := range addressList {
		go func(addr string) {
			conn, err := grpc.Dial(
				addr+":"+port,
				grpc.WithInsecure(),
				grpc.WithTimeout(5*time.Second), // Increase connection timeout
			)
			if err != nil {
				results <- Result{addr, nil, err}
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
				results <- Result{addr, nil, err}
				return
			}

			// Clean invalid UTF-8 strings
			for _, rankLog := range resp.Ranklogs {
				for _, entry := range rankLog.Entries {
					entry.Message = utils.CleanUTF8(entry.Message)
				}
			}

			results <- Result{addr, resp, nil}
		}(addr)
	}
	list := []string{}
	// Collect results
	finalResponse := &pb.LogResponse{}
	for i := 0; i < len(addressList); i++ {
		res := <-results
		if res.err != nil {
			fmt.Printf("Failed to get logs from node %s: %v\n", res.node, res.err)
			continue
		}
		if rank == "" {
			// If rank is empty, add all logs
			finalResponse.Ranklogs = append(finalResponse.Ranklogs, res.logs.Ranklogs...)
		} else {
			// If rank is not empty, only add logs for the specified rank
			for _, rankLog := range res.logs.Ranklogs {

				if rankLog.Rank == rank {
					finalResponse.Ranklogs = append(finalResponse.Ranklogs, rankLog)
				}
			}
		}
	}
	close(results)
	maxRank := ""
	for _, s := range list {
		if maxRank == "" || utils.CompareRank(s, maxRank) > 0 {
			maxRank = s
		}
	}

	// If list is not empty, check if maximum value is less than target
	allLess := len(list) == 0 || utils.CompareRank(maxRank, rank) < 0
	if allLess && maxRank != "" {
		fmt.Println("Please enter a valid rank")
		return
	}

	customResponse := CustomLogResponse{}
	for _, rankLog := range finalResponse.Ranklogs {
		var customEntries []CustomLogEntry
		for _, entry := range rankLog.Entries {
			customEntries = append(customEntries, CustomLogEntry{
				Timestamp: utils.FormatTimestamp(entry.Timestamp),
				Level:     entry.Level,
				Epoch:     entry.Epoch,
				Message:   entry.Message,
			})
		}
		customResponse.Ranklogs = append(customResponse.Ranklogs, CustomRankLog{
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
			return
		}
		if rank == "" {
			rank = "allrank"
		}
		fileName := fmt.Sprintf("%s_logs_%s.json", jobName, rank)
		err = utils.AppendWithTimestamp("logs", fileName, jsonData)
		fmt.Println(string(jsonData))
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("Log data successfully saved to %s\n", fileName)
		}
	}
}
