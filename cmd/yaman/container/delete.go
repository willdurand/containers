package container

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:     "delete <container> [<container>...]",
		Aliases: []string{"del", "rm", "remove"},
		Short:   "Delete one or more containers",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			all, _ := cmd.Flags().GetBool("all")

			if !all && len(args) < 1 {
				return fmt.Errorf("requires at least 1 arg(s), only received %d", len(args))
			}

			return nil
		},
		Run:               cli.HandleErrors(delete),
		Args:              cobra.MinimumNArgs(0),
		ValidArgsFunction: completeManyContainerIds,
	}
	cmd.Flags().BoolP("all", "a", false, "delete all stopped containers")
	containerCommand.AddCommand(cmd)
}

func delete(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	all, _ := cmd.Flags().GetBool("all")

	if all {
		args = yaman.GetContainerIds(rootDir, "")
	}

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
