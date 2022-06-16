package container

import (
	"encoding/json"
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "inspect <container>",
		Short:             "Return low-level information on the container as JSON",
		RunE:              inspect,
		SilenceUsage:      true,
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func inspect(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	container, err := yaman.Inspect(rootDir, args[0])
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(container); err != nil {
		return err
	}

	return nil
}
