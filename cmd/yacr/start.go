package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	cmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Start a container",
		Run:   cli.HandleErrors(start),
		Args:  cobra.ExactArgs(1),
	}
	rootCmd.AddCommand(cmd)
}

func start(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	if err := yacr.Start(rootDir, args[0]); err != nil {
		return fmt.Errorf("start: %w", err)
	}

	return nil
}
