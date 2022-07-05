package main

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm"
)

func init() {
	startCmd := &cobra.Command{
		Use:   "start <id>",
		Short: "Start a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			return microvm.Start(rootDir, args[0])
		}),
		Args: cobra.ExactArgs(1),
	}
	rootCmd.AddCommand(startCmd)
}
