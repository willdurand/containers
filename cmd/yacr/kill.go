package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	cmd := &cobra.Command{
		Use:          "kill <id> [<signal>]",
		Short:        "Send a signal to a container",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			if err := yacr.Kill(rootDir, args); err != nil {
				return fmt.Errorf("kill: %w", err)
			}

			return nil
		},
		Args: cobra.MinimumNArgs(1),
	}
	cmd.Flags().Bool("all", false, "UNSUPPORTED FLAG")

	rootCmd.AddCommand(cmd)
}
