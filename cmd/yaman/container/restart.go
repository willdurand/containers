package container

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "restart <container>",
		Short:             "Restart a container",
		Run:               cli.HandleErrors(restart),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func restart(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	return yaman.Restart(rootDir, args[0])
}
