// Copyright (c) OpenMMLab. All rights reserved.

package alerts

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"time"

	"deeptrace/pkg/client/utils"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// NodeAlerts contains alert information and processing results for a single node
type NodeAlerts struct {
	NodeAddr string
	Alerts   []*pb.AlertRecord
	Error    error
}

func NewCmdAlerts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alerts",
		Short: "Get alert information",
		Long: `Get alert information for the specified job.
Support filtering by time range and severity level.

Usage:
  client alerts --job-id <job name> -a address_file [--interval-alert <interval minutes>][--min-severity <minimum severity level>] [--port <server port>]

Severity levels: INFO, WARNING, ERROR, CRITICAL

Example:
  client alerts --job-id my_job -a address_file --interval-alert 2 --min-severity WARNING  # Get WARNING level and above alerts every 2 minutes
`,
		Run: func(cmd *cobra.Command, args []string) {
			// Get job name
			jobName, _ := cmd.Flags().GetString("job-id")
			if jobName == "" {
				jobName = viper.GetString("job-id")
			}
			if jobName == "" {
				fmt.Println("Error: Job name must be specified")
				os.Exit(1)
			}
			fmt.Printf("Using job name: %s\n", jobName)

			// Get address list file path
			addressListPath, _ := cmd.Flags().GetString("address-list")
			if addressListPath == "" {
				addressListPath = viper.GetString("address-list")
			}
			if addressListPath == "" {
				fmt.Println("Error: Agent address list file path must be specified")
				os.Exit(1)
			}
			// Read address list
			addressList, err := utils.ReadAddressListFromFile(addressListPath)
			if err != nil {
				fmt.Printf("Failed to read address list file: %v\n", err)
				os.Exit(1)
			}
			fmt.Printf("Obtained addresses: %v\n", addressList)

			// Get port number
			port, _ := cmd.Flags().GetString("port")
			if port == "" {
				port = viper.GetString("port")
				if port != "" {
					fmt.Printf("Using port number specified in configuration file: %s\n", port)
				} else {
					fmt.Println("Port number not specified, using default value 50051")
					port = "50051"
				}
			} else {
				fmt.Printf("Using port number specified on command line: %s\n", port)
			}

			// Get time interval (default 0 means no loop)
			intervalMinutes, _ := cmd.Flags().GetInt("interval-alert")
			if intervalMinutes < 0 {
				fmt.Println("Error: Time interval cannot be negative")
				os.Exit(1)
			} else if intervalMinutes == 0 {
				intervalMinutes = viper.GetInt("interval-alert")
				if intervalMinutes == 0 {
					fmt.Println("interval-alert is empty in configuration file, using default time interval of 1 minute")
					intervalMinutes = 1
				}
			}

			// Get minimum severity level
			minSeverityStr, _ := cmd.Flags().GetString("min-severity")
			if minSeverityStr == "" {
				minSeverityStr = viper.GetString("min-severity")
			}

			var minSeverity pb.Severity
			switch minSeverityStr {
			case "INFO":
				minSeverity = pb.Severity_INFO
			case "WARNING":
				minSeverity = pb.Severity_WARNING
			case "ERROR":
				minSeverity = pb.Severity_ERROR
			case "CRITICAL":
				minSeverity = pb.Severity_CRITICAL
			case "":
				minSeverity = pb.Severity_INFO
				fmt.Println("Minimum severity level not specified, defaulting to INFO")
			default:
				fmt.Printf("Error: Invalid severity level: %s\n", minSeverityStr)
				os.Exit(1)
			}
			fmt.Printf("Minimum severity level: %s\n", minSeverity.String())
			isFRUN := true
			// Define function to execute single alert check
			var lastEndTime *time.Time // Record end time of last check
			checkAlerts := func() {
				// Calculate time range for this execution (if interval is set)
				var startTime, endTime *time.Time
				now := time.Now()
				endTime = &now

				if intervalMinutes > 0 {
					// If interval is set, automatically calculate time range (from last end time to now)
					if lastEndTime != nil {
						startTime = lastEndTime
					} else {
						// First run, calculate time from interval minutes ago
						st := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.Local) // UTC timezone
						startTime = &st
					}
					if !isFRUN {
						fmt.Printf("==== Executing periodic check: %s to %s ====\n",
							startTime.Local().Format("2006-01-02 15:04:05"),
							endTime.Local().Format("2006-01-02 15:04:05"))
					} else {
						fmt.Printf("==== Executing initial check to %s ====\n",
							endTime.Local().Format("2006-01-02 15:04:05"))
						isFRUN = false
					}
				} else {
					// Interval not set, use time range specified on command line
					startTimeStr, _ := cmd.Flags().GetString("start-time")
					endTimeStr, _ := cmd.Flags().GetString("end-time")

					if startTimeStr != "" {
						st, err := time.Parse(time.RFC3339, startTimeStr)
						if err != nil {
							localSt, localErr := time.Parse("2006-01-02T15:04:05", startTimeStr)
							if localErr != nil {
								fmt.Printf("Error: Invalid start time format: %v\n", err)
								return
							}
							st = localSt
						}
						startTime = &st
						fmt.Printf("Start time: %s (local time)\n", st.Local().Format("2006-01-02 15:04:05"))
					} else {
						fmt.Println("Start time not specified, will get earliest alerts")
					}

					if endTimeStr != "" {
						et, err := time.Parse(time.RFC3339, endTimeStr)
						if err != nil {
							localEt, localErr := time.Parse("2006-01-02T15:04:05", endTimeStr)
							if localErr != nil {
								fmt.Printf("Error: Invalid end time format: %v\n", err)
								return
							}
							et = localEt
						}
						endTime = &et
						fmt.Printf("End time: %s (local time)\n", et.Local().Format("2006-01-02 15:04:05"))
					} else {
						fmt.Println("End time not specified, will get latest alerts")
					}
				}

				fmt.Printf("Found %d addresses: %v\n", len(addressList), addressList)

				// Call alert service client
				results := GetAlerts(addressList, port, startTime, endTime, minSeverity)
				noAlert := PrintAllNodeAlerts(results)
				feishuWebhookURL := viper.GetString("FS-URL")
				// Send alerts to Feishu
				if feishuWebhookURL != "" {
					err := SendAlertsToFeishu(feishuWebhookURL, results, "Alert Notification:", noAlert)
					if err != nil {
						fmt.Printf("Failed to send Feishu alert: %v\n", err)
					}
				} else {
					fmt.Println("Feishu link not specified, not sending Feishu alert")
				}

				// Update last end time
				lastEndTime = endTime
			}

			// Execute immediately
			checkAlerts()

			// If interval is set, start timer to execute in a loop
			if intervalMinutes > 0 {
				interval := time.Duration(intervalMinutes) * time.Minute
				ticker := time.NewTicker(interval)
				fmt.Printf("Alert checks will be executed every %d minutes...\n", intervalMinutes)

				// Handle interrupt signal
				done := make(chan bool)
				go func() {
					sig := make(chan os.Signal, 1)
					signal.Notify(sig, os.Interrupt)
					<-sig
					done <- true
				}()

				// Timer loop
				for {
					select {
					case <-ticker.C:
						checkAlerts()
					case <-done:
						ticker.Stop()
						fmt.Println("\nProgram stopped")
						return
					}
				}
			}
		},
	}

	// Add command line flags
	cmd.Flags().IntP("interval-alert", "i", 0, "Automatic execution interval (minutes), 0 means execute once only")
	cmd.Flags().StringP("start-time", "s", "", "Start time (format: YYYY-MM-DDTHH:MM:SS[Z|±HH:MM])")
	_ = cmd.Flags().MarkHidden("start-time")
	cmd.Flags().StringP("end-time", "e", "", "End time (format: YYYY-MM-DDTHH:MM:SS[Z|±HH:MM])")
	_ = cmd.Flags().MarkHidden("end-time")
	cmd.Flags().StringP("min-severity", "m", "", "Minimum severity level (INFO, WARNING, ERROR, CRITICAL)")
	return cmd
}

// GetAlerts concurrently gets alert information
// Parameter description:
//   - addrs: Address list (without port)
//   - port: gRPC service port
//   - startTime: Start time (optional, nil means no limit)
//   - endTime: End time (optional, nil means no limit)
//   - minSeverity: Minimum severity level
func GetAlerts(addrs []string, port string, startTime, endTime *time.Time, minSeverity pb.Severity) []NodeAlerts {
	var wg sync.WaitGroup
	resultCh := make(chan NodeAlerts, len(addrs))

	// Concurrently process each node
	for _, addr := range addrs {
		wg.Add(1)
		go func(address string) {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Create connection
			conn, err := grpc.Dial(
				address+":"+port,
				grpc.WithInsecure(),
				grpc.WithTimeout(5*time.Second),
			)
			if err != nil {
				resultCh <- NodeAlerts{
					NodeAddr: address,
					Error:    fmt.Errorf("Failed to connect to node %s: %v", address, err),
				}
				return
			}
			defer conn.Close()

			// Create client
			client := pb.NewAlertServiceClient(conn)

			// Build request (convert time.Time to google.protobuf.Timestamp)
			req := &pb.GetAlertsRequest{
				Unprocessed: true,
				MinSeverity: minSeverity,
			}
			// Only set when valid time is passed (nil means no limit)
			if startTime != nil {
				req.StartTime = timestamppb.New(*startTime)
			}
			if endTime != nil {
				req.EndTime = timestamppb.New(*endTime)
			}

			fmt.Printf("Getting alert information from node %s...\n", address)

			// Send request
			resp, err := client.GetAlerts(ctx, req)
			if err != nil {
				resultCh <- NodeAlerts{
					NodeAddr: address,
					Error:    fmt.Errorf("failed to get alerts for node %s: %v", address, err),
				}
				return
			}

			resultCh <- NodeAlerts{
				NodeAddr: address,
				Alerts:   resp.Alerts,
			}
		}(addr)
	}

	// Wait for all nodes to finish processing
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results
	var results []NodeAlerts
	for result := range resultCh {
		results = append(results, result)
	}

	return results
}

// PrintAllNodeAlerts prints alert information for all nodes (adapted for google.protobuf.Timestamp)
func PrintAllNodeAlerts(results []NodeAlerts) []bool {
	noAlert := make([]bool, len(results))
	for idx, result := range results {

		fmt.Printf("==== Alert information from node %s ====\n", result.NodeAddr)
		if result.Error != nil {
			fmt.Printf("Error: %v\n", result.Error)
			continue
		}

		if len(result.Alerts) == 0 {
			fmt.Println("No matching alert information found")
			noAlert[idx] = true
			continue
		}

		for i, alert := range result.Alerts {
			// Convert google.protobuf.Timestamp to time.Time (handle nil safely)
			var t time.Time
			if alert.Timestamp != nil {
				t = alert.Timestamp.AsTime()
			} else {
				t = time.Time{} // Empty time
			}

			fmt.Printf("[%d] Time: %s | Level: %s | Message: %s\n",
				i+1,
				t.Local().Format("2006-01-02 15:04:05.000"), // millisecond
				alert.Severity.String(),
				alert.Message)
		}
		fmt.Println()

	}
	return noAlert
}
