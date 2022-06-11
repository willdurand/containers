package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	cmd := &cobra.Command{
		Use:          "create <id>",
		Short:        "Create a container",
		SilenceUsage: true,
		RunE:         create,
		Args:         cobra.ExactArgs(1),
	}
	cmd.PersistentFlags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	cmd.MarkFlagRequired("bundle")
	cmd.Flags().String("pid-file", "", "specify the file to write the process id to")
	cmd.Flags().String("console-socket", "", "console unix socket used to pass a PTY descriptor")
	cmd.Flags().Bool("no-pivot", false, "do not use pivot root to jail process inside rootfs")
	rootCmd.AddCommand(cmd)

	containerCmd := &cobra.Command{
		Use:          "container <id>",
		SilenceUsage: true,
		RunE:         createContainer,
		Hidden:       true,
		Args:         cobra.ExactArgs(1),
	}
	cmd.AddCommand(containerCmd)
}

func create(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	bundle, _ := cmd.Flags().GetString("bundle")
	pidFile, _ := cmd.Flags().GetString("pid-file")
	consoleSocket, _ := cmd.Flags().GetString("console-socket")
	noPivot, _ := cmd.Flags().GetBool("no-pivot")
	logFile, _ := cmd.Flags().GetString("log")
	logFormat, _ := cmd.Flags().GetString("log-format")
	debug, _ := cmd.Flags().GetBool("debug")

	opts := yacr.CreateOpts{
		ID:            args[0],
		Bundle:        bundle,
		PidFile:       pidFile,
		ConsoleSocket: consoleSocket,
		NoPivot:       noPivot,
		LogFile:       logFile,
		LogFormat:     logFormat,
		Debug:         debug,
	}

	if err := yacr.Create(rootDir, opts); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	return nil
}
