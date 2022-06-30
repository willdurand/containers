package container

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "stop <container> [<container>...]",
		Short:             "Stop one or more containers",
		Run:               cli.HandleErrors(stop),
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func stop(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	for _, id := range args {
		if err := yaman.Stop(rootDir, id); err != nil {
			logrus.WithFields(logrus.Fields{
				"id":    id,
				"error": err,
			}).Debug("failed to delete container")
			cli.PrintUserError(err)
		}
	}
	return nil
}
