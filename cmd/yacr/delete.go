package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	cmd := &cobra.Command{
		Use:          "delete <id>",
		Aliases:      []string{"del", "rm"},
		Short:        "Delete a container",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")
			force, _ := cmd.Flags().GetBool("force")

			if err := yacr.Delete(rootDir, args[0], force); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			return nil
		},
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().BoolP("force", "f", false, "force delete a container")

	rootCmd.AddCommand(cmd)
}
