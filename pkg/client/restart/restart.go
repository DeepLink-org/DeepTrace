// Copyright (c) OpenMMLab. All rights reserved.

package restart

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"deeptrace/pkg/client/utils"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

func NewCmdRestart() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "restart",
		Short: "Restart agent on a node",
		Long: `Restart agent on a node.
Usage:
  client restart --job-id -w clusterx <job name> [--port <server port>]

Examples:
  client restart --job-id my_job -w clusterx --port 50052`,
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

			AuthToken := ""
			RestartAgent(addressList, AuthToken, port)
		},
	}

	return cmd
}

func RestartAgent(addressList []string, authToken string, port string) {
	var wg sync.WaitGroup
	errorCh := make(chan error, len(addressList))

	// Concurrently process each node (can be adjusted to sequential execution as needed)
	for _, addr := range addressList {
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
				errorCh <- fmt.Errorf("Failed to connect to node %s: %v", address, err)
				return
			}
			defer conn.Close()

			// Create client
			client := pb.NewDeepTraceServiceClient(conn)

			// Send restart request
			req := &pb.RestartRequest{AuthToken: authToken}
			fmt.Printf("Requesting restart of node %s...\n", address)

			resp, err := client.RestartServer(ctx, req)
			if err != nil {
				errorCh <- fmt.Errorf("Request to node %s failed: %v", address, err)
				return
			}

			// Handle response
			if resp.Success {
				fmt.Printf("Node %s restarted successfully: %s\n", address, resp.Message)
			} else {
				errorCh <- fmt.Errorf("Node %s restart failed: %s", address, resp.Message)
			}
		}(addr)
	}
	// Wait for all nodes to finish processing
	wg.Wait()
	close(errorCh)

	// Summarize error information
	var errors []string
	for err := range errorCh {
		errors = append(errors, err.Error())
	}

	if len(errors) > 0 {
		fmt.Printf("Encountered %d errors during restart:\n", len(errors))
		for _, err := range errors {
			fmt.Println("-", err)
		}
	} else {
		fmt.Println("Restart requests successfully sent to all nodes")
	}
}
