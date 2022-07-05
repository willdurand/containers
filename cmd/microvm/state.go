package main

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm"
)

func init() {
	stateCmd := &cobra.Command{
		Use:   "state <id>",
		Short: "Query the state of a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			return microvm.State(rootDir, args[0], os.Stdout)
		}),
		Args: cobra.ExactArgs(1),
	}
	rootCmd.AddCommand(stateCmd)
}
