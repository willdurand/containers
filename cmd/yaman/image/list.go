package image

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/docker/go-units"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List images",
		RunE:    list,
		Args:    cobra.NoArgs,
	}
	imageCommand.AddCommand(cmd)
}

func list(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	list, err := yaman.ListImages(rootDir)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
	fmt.Fprint(w, "NAME\tTAG\tCREATED\tPULLED\tREGISTRY\n")

	for _, img := range list {
		fmt.Fprintf(
			w, "%s\t%s\t%s\t%s ago\t%s\n",
			img.Name,
			img.Version,
			img.Created.Format(time.RFC3339),
			units.HumanDuration(time.Since(img.Pulled)),
			img.Registry,
		)
	}

	return w.Flush()
}
