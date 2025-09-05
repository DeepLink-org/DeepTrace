// Copyright (c) OpenMMLab. All rights reserved.

package version

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"deeptrace/pkg/client/utils"
	v "deeptrace/pkg/version"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type VersionResult struct {
	Address string
	Resp    *pb.VersionResponse
	Err     error
	Status  string // "success" or "error"
}

// Add return value type, including version collection and error information
type AgentInfoResult struct {
	UniqueVersions map[string]struct{}
	Errors         []string
}

func NewCmdVersion() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Get client and agent related information",
		Long: `Get client and agent related information.
Usage:
  client version --job-id <job name> -a address_file [--port <service port>]

Example:
  client version --job-id my_job -a address_file --port 50052`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(v.GetClientVersionInfo())
			jobName, _ := cmd.Flags().GetString("job-id")
			if jobName == "" {
				jobName = viper.GetString("job-id")
			}
			if jobName != "" {
				fmt.Printf("Using job name: %s\n", jobName)
			} else {
				fmt.Println("Note: Job name not specified")
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

			result := GetAgentInfo(addressList, port)

			// Print all different versions
			if len(result.UniqueVersions) > 1 {
				fmt.Print("Detected different versions: ")
				for ver := range result.UniqueVersions {
					fmt.Printf("%s ", ver)
				}
				fmt.Println()
			}
			if len(result.UniqueVersions) == 0 {
				fmt.Println("No version information obtained")
			}
		},
	}
	return cmd
}

func GetAgentInfo(addressList []string, port string) AgentInfoResult {
	var wg sync.WaitGroup
	versions := make(map[string]struct{}) // Used to track unique versions
	// Create a buffered result channel (buffer size equals number of nodes)
	resultCh := make(chan VersionResult, len(addressList))

	// Start goroutines to process each node
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
				grpc.WithTimeout(5*time.Second), // Increase connection timeout
			)
			if err != nil {
				resultCh <- VersionResult{
					Address: address,
					Err:     err,
					Status:  "error",
				}
				return
			}
			defer conn.Close()

			// Create client and send request
			client := pb.NewDeepTraceServiceClient(conn)
			resp, err := client.GetVersion(ctx, &emptypb.Empty{})

			// Send result to channel
			if err != nil {
				resultCh <- VersionResult{
					Address: address,
					Err:     err,
					Status:  "error",
				}
			} else {
				resultCh <- VersionResult{
					Address: address,
					Resp:    resp,
					Status:  "success",
				}
			}
		}(addr)
	}

	// Close channel after all goroutines complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Main goroutine reads results from channel and prints (natural serialization)
	var errors []string

	for res := range resultCh {
		if res.Status == "error" {
			errors = append(errors, fmt.Sprintf("Node %s: %v", res.Address, res.Err))
			continue
		}

		// Print successful results
		fmt.Printf("Version information for agent on node %s:\n", res.Address)
		fmt.Printf("  - Version: %s\n", res.Resp.Version)
		fmt.Printf("  - Commit: %s\n", res.Resp.Commit)
		fmt.Printf("  - Build Time: %s\n", res.Resp.BuildTime)
		fmt.Printf("  - Build Tag: %s\n", res.Resp.BuildTag)
		fmt.Println()
		// Record version
		versions[res.Resp.Version] = struct{}{}
	}

	// Print error information
	if len(errors) > 0 {
		fmt.Printf("Encountered %d errors during processing:\n", len(errors))
		for _, err := range errors {
			fmt.Println("-", err)
		}
	}
	return AgentInfoResult{
		UniqueVersions: versions,
		Errors:         errors,
	}
}
