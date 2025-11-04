// Copyright (c) OpenMMLab. All rights reserved.

package client

import (
	"fmt"

	"deeptrace/pkg/client/alerts"
	"deeptrace/pkg/client/checkhang"
	"deeptrace/pkg/client/logs"
	"deeptrace/pkg/client/restart"
	"deeptrace/pkg/client/stacks"
	"deeptrace/pkg/client/version"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// readConfig reads parameters from the configuration file
func readConfig(configPath string) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		fmt.Println("Note: User did not specify configuration file path, defaulting to deeptracex.yaml in this directory")
		viper.SetConfigName("deeptracex")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
	}
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("Error reading configuration file: Using default values or user-specified values\n")
	}
}

func NewDeepTracexCommand() *cobra.Command {
	// Read configuration file
	var configPath string

	// Create root command
	cmds := &cobra.Command{
		Use:   "deeptracex",
		Short: "Command line tool",
		Long: `This is a distributed training task diagnostic tool.
Usage:
  deeptracex [subcommand] [parameters]

Example:
  deeptracex logs --job-id my_job -w clusterx --port 50051`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			readConfig(configPath)
		},
	}

	// Disable auto-completion command
	cmds.CompletionOptions.DisableDefaultCmd = true

	// Add global flags
	cmds.PersistentFlags().StringP("job-id", "j", "", "Specify job name")
	cmds.PersistentFlags().StringP("port", "p", "", "Specify service port number")
	//rootCmd.PersistentFlags().StringP("username", "u", "", "Specify username")
	cmds.PersistentFlags().StringVarP(&configPath, "config", "c", "", "Specify the path to the configuration file")
	cmds.PersistentFlags().StringP("worker-source", "w", "", "Specify the workers of your job: clusterx or path to the file contains workers")

	// Add subcommands directly to the root command
	cmds.AddCommand(
		logs.NewCmdLogs(),
		stacks.NewCmdStacks(),
		checkhang.NewCmdCheckHang(),
		restart.NewCmdRestart(),
		version.NewCmdVersion(),
		alerts.NewCmdAlerts(),
	)

	return cmds
}
