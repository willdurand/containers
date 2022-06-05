package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacr/containers"
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
				files, err := ioutil.ReadDir(rootDir)
				if err != nil {
					return fmt.Errorf("list: failed to read root directory: %w", err)
				}

				w := tabwriter.NewWriter(os.Stdout, 12, 1, 3, ' ', 0)
				fmt.Fprint(w, "ID\tSTATUS\tCREATED\tPID\tBUNDLE\n")

				for _, f := range files {
					if !f.IsDir() {
						continue
					}

					container, err := containers.Load(rootDir, f.Name())
					if err != nil {
						continue
					}

					state := container.State()

					pid := state.Pid
					if container.IsStopped() {
						pid = 0
					}

					fmt.Fprintf(
						w, "%s\t%s\t%s\t%d\t%s\n",
						container.ID(),
						state.Status,
						container.CreatedAt().Format(time.RFC3339),
						pid,
						state.Bundle,
					)
				}

				return w.Flush()
			},
			Args: cobra.NoArgs,
		},
	)
}
