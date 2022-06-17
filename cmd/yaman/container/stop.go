package container

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "stop <container>",
		Short:             "Stop a container",
		Run:               cli.HandleErrors(stop),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func stop(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	return yaman.Stop(rootDir, args[0])
}
