package container

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:          "cleanup <container>",
		Short:        "Clean-up a container",
		Hidden:       true,
		RunE:         cleanUp,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}
	containerCommand.AddCommand(cmd)
}

func cleanUp(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	return yaman.CleanUp(rootDir, args[0])
}
