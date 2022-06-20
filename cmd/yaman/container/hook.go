package container

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:    "hook <hook>",
		Short:  "Hidden command called by the OCI runtime",
		Hidden: true,
		Run:    cli.HandleErrors(hook),
		Args:   cobra.ExactArgs(1),
	}
	containerCommand.AddCommand(cmd)
}

func hook(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	return yaman.ProcessHook(rootDir, args[0], os.Stdin)
}
