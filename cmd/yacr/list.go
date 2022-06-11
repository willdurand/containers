package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:          "list",
			Aliases:      []string{"ls"},
			Short:        "List containers",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				rootDir, _ := cmd.Flags().GetString("root")

				list, err := yacr.List(rootDir)
				if err != nil {
					return fmt.Errorf("list: %w", err)
				}

				w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
				fmt.Fprint(w, "ID\tSTATUS\tCREATED\tPID\tBUNDLE\n")

				for _, container := range list {
					fmt.Fprintf(
						w, "%s\t%s\t%s\t%d\t%s\n",
						container.ID,
						container.Status,
						container.CreatedAt.Format(time.RFC3339),
						container.PID,
						container.BundlePath,
					)
				}

				return w.Flush()
			},
			Args: cobra.NoArgs,
		},
	)
}
