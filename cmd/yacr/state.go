package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:          "state <id>",
			Short:        "Query the state of a container",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				rootDir, _ := cmd.Flags().GetString("root")

				if err := yacr.WriteState(rootDir, args[0], os.Stdout); err != nil {
					return fmt.Errorf("state: %w", err)
				}

				return nil
			},
			Args: cobra.ExactArgs(1),
		},
	)
}
