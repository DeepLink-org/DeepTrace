// Copyright (c) OpenMMLab. All rights reserved.

package stacks

import (
	"context"
	"fmt"
	"os"
	"time"

	"deeptrace/pkg/client/utils"
	pb "deeptrace/v1"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/encoding/protojson"
)

// NewCmdStacks creates a cobra command for fetching stack information via gRPC
func NewCmdStacks() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "stacks",
		Short: "Get information through stack",
		Long: `Get stack information for the specified job.
Usage:
  client stacks --job-id <job name> -w clusterx --process-type <process type> --rank <rank> [--port <service port>]

Example:
  client stacks --job-id my_job -w clusterx --process-type PROCESS_TRAINER --rank 0 --port 50052`,
		Run: func(cmd *cobra.Command, args []string) {
			jobName, _ := cmd.Flags().GetString("job-id")
			if jobName == "" {
				jobName = viper.GetString("job-id")
			}
			if jobName != "" {
				fmt.Printf("Using job name: %s\n", jobName)
			} else {
				fmt.Println("Note: Job name not specified")
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

			processTypeStr, _ := cmd.Flags().GetString("process-type")
			rank, _ := cmd.Flags().GetString("rank")
			port, _ := cmd.Flags().GetString("port")
			fmt.Printf("Input rank: %s\n", rank)
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
			processType := pb.ProcessType_PROCESS_UNSPECIFIED

			switch processTypeStr {
			case "PROCESS_TRAINER":
				processType = pb.ProcessType_PROCESS_TRAINER
			case "PROCESS_DATA_LOADER":
				processType = pb.ProcessType_PROCESS_DATA_LOADER
			case "PROCESS_LAUNCHER":
				processType = pb.ProcessType_PROCESS_LAUNCHER
			case "":
				processTypeStr = viper.GetString("process-type")
				switch processTypeStr {
				case "PROCESS_TRAINER":
					processType = pb.ProcessType_PROCESS_TRAINER
				case "PROCESS_DATA_LOADER":
					processType = pb.ProcessType_PROCESS_DATA_LOADER
				case "PROCESS_LAUNCHER":
					processType = pb.ProcessType_PROCESS_LAUNCHER
				default:
					processType = pb.ProcessType_PROCESS_UNSPECIFIED
				}
			default:
				fmt.Printf("Error: Invalid process type %q, available types: PROCESS_TRAINER, PROCESS_DATA_LOADER, PROCESS_LAUNCHER\n", processTypeStr)
				os.Exit(1)
			}

			FetchStacksFromNodes(jobName, addressList, processType, rank, port)
		},
	}

	cmd.Flags().String("process-type", "", "Target process type (PROCESS_TRAINER, PROCESS_DATA_LOADER, PROCESS_LAUNCHER, if not specified, return all types)")
	cmd.Flags().String("rank", "", "Rank number, if not specified, return all ranks")

	return cmd
}

func FetchStacksFromNodes(jobName string, addressList []string, processType pb.ProcessType, rank string, port string) {
	type Result struct {
		address string
		stacks  *pb.ProcessStacksResponse
		err     error
	}

	results := make(chan Result, len(addressList))

	// Start goroutines to process each node in parallel
	for _, addr := range addressList {
		go func(address string) {
			conn, err := grpc.Dial(
				address+":"+port,
				grpc.WithInsecure(),
				grpc.WithTimeout(5*time.Second), // Increase connection timeout
			)
			if err != nil {
				results <- Result{address, nil, err}
				return
			}
			defer conn.Close()

			client := pb.NewDeepTraceServiceClient(conn)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			req := &pb.GetProcessStacksRequest{
				ProcessType: processType,
				Rank:        rank,
			}

			resp, err := client.GetProcessStacks(ctx, req)
			if err != nil {
				results <- Result{address, nil, err}
				return
			}

			results <- Result{address, resp, nil}
		}(addr)
	}

	// Collect all ProcessInfo
	var allProcesses []*pb.ProcessInfo
	for i := 0; i < len(addressList); i++ {
		res := <-results
		if res.err != nil {
			fmt.Printf("Failed to get stack information from node %s: %v\n", res.address, res.err)
			continue
		}
		allProcesses = append(allProcesses, res.stacks.Processes...)
	}
	close(results)

	// Convert results to JSON and output
	jsonMarshal := protojson.MarshalOptions{
		UseEnumNumbers: false,
		Multiline:      true,
		Indent:         " ",
	}
	data, err := jsonMarshal.Marshal(&pb.ProcessInfoList{
		Processes:  allProcesses,
		TotalCount: int32(len(allProcesses)),
	})
	if err != nil {
		fmt.Printf("Failed to convert to JSON: %v\n", err)
		return
	}
	if rank == "" {
		rank = "allrank"
	}
	fileName := fmt.Sprintf("%s_stack_%s.json", jobName, rank)

	if len(data) > 2 {
		err = utils.AppendWithTimestamp("stacks", fileName, data)
		fmt.Println(string(data))
		if err != nil {
			fmt.Println("Error:", err)
		} else {
			fmt.Printf("Process data successfully saved to %s\n", fileName)
		}
	}

}
