package container

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List containers",
		RunE:  list,
		Args:  cobra.NoArgs,
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
	fmt.Fprint(w, "CONTAINER ID\tIMAGE\tCOMMAND\tSTATUS\tNAME\n")

	for _, container := range list {
		fmt.Fprintf(
			w, "%s\t%s\t%s\t%s\t%s\n",
			container.ID,
			container.Image,
			container.Command,
			container.Status,
			container.Name,
		)
	}

	return w.Flush()
}
