package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacr/containers"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:          "state <id>",
			Short:        "Query the state of a container",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				rootDir, _ := cmd.Flags().GetString("root")
				container, err := containers.Load(rootDir, args[0])
				if err != nil {
					return fmt.Errorf("state: %w", err)
				}

				if err := json.NewEncoder(os.Stdout).Encode(container.State()); err != nil {
					return fmt.Errorf("state: %w", err)
				}

				return nil
			},
			Args: cobra.ExactArgs(1),
		},
	)
}
