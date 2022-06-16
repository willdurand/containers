package container

import (
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "delete <container> [<container>...]",
		Aliases:           []string{"del", "rm", "remove"},
		Short:             "Delete one or more containers",
		RunE:              delete,
		SilenceUsage:      true,
		Args:              cobra.MinimumNArgs(1),
		ValidArgsFunction: completeManyContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func delete(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	for _, id := range args {
		if err := yaman.Delete(rootDir, id); err != nil {
			logrus.WithFields(logrus.Fields{
				"id":    id,
				"error": err,
			}).Debug("failed to delete container")
			cli.PrintUserError(err)
		}
	}

	return nil
}
