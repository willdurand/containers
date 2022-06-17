package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	cmd := &cobra.Command{
		Use:   "state <id>",
		Short: "Query the state of a container",
		Run:   cli.HandleErrors(state),
		Args:  cobra.ExactArgs(1),
	}
	rootCmd.AddCommand(cmd)
}

func state(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	if err := yacr.State(rootDir, args[0], os.Stdout); err != nil {
		return fmt.Errorf("state: %w", err)
	}

	return nil
}
