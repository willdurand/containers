package container

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:       "hook <hook>",
		Short:     "Hidden command called by the OCI runtime",
		Hidden:    true,
		Run:       cli.HandleErrors(hook),
		Args:      cobra.ExactArgs(1),
		ValidArgs: []string{"CreateRuntime"},
	}
	containerCommand.AddCommand(cmd)
}

func hook(cmd *cobra.Command, args []string) error {
	return yaman.ProcessHook(os.Stdin, args[0])
}
