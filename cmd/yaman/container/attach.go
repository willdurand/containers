package container

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "attach <container>",
		Short:             "Attach standard input, output, and error streams to a running container",
		Run:               cli.HandleErrors(attach),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	containerCommand.AddCommand(cmd)
}

func attach(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	opts := yaman.AttachOpts{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}

	return yaman.Attach(rootDir, args[0], opts)
}
