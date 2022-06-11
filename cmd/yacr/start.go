package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:          "start <id>",
			Short:        "Start a container",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				rootDir, _ := cmd.Flags().GetString("root")

				if err := yacr.Start(rootDir, args[0]); err != nil {
					return fmt.Errorf("start: %w", err)
				}

				return nil
			},
			Args: cobra.ExactArgs(1),
		},
	)
}
