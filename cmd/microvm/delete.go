package main

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
)

func init() {
	deleteCmd := &cobra.Command{
		Use:     "delete <id>",
		Aliases: []string{"del", "rm"},
		Short:   "Delete a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")
			baseDir := filepath.Join(rootDir, args[0])

			return os.RemoveAll(baseDir)
		}),
		Args: cobra.ExactArgs(1),
	}
	deleteCmd.Flags().BoolP("force", "f", false, "UNSUPPORTED FLAG")
	rootCmd.AddCommand(deleteCmd)
}
