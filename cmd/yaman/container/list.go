package container

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List containers",
		Aliases: []string{"ls"},
		Run:     cli.HandleErrors(list),
		Args:    cobra.NoArgs,
	}
	cmd.Flags().BoolP("all", "a", false, "show all containers and not just those running")
	containerCommand.AddCommand(cmd)
}

func list(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	all, _ := cmd.Flags().GetBool("all")

	list, err := yaman.ListContainers(rootDir, all)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "CONTAINER ID\tIMAGE\tCOMMAND\tCREATED\tSTATUS\tPORTS\n")

	for _, container := range list {
		exposedPorts := fmt.Sprint(container.ExposedPorts)
		exposedPorts = exposedPorts[1 : len(exposedPorts)-1]

		fmt.Fprintf(
			w, "%s\t%s\t%s\t%s ago\t%s\t%s\n",
			container.ID,
			container.Image,
			container.Command,
			units.HumanDuration(time.Since(container.Created)),
			container.Status,
			exposedPorts,
		)
	}

	return w.Flush()
}
