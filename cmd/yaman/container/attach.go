package container

import (
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
	cmd.Flags().Bool("no-stdin", false, "do not attach stdin")
	containerCommand.AddCommand(cmd)
}

func attach(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	noStdin, _ := cmd.Flags().GetBool("no-stdin")

	opts := yaman.AttachOpts{
		Stdin:  !noStdin,
		Stdout: true,
		Stderr: true,
	}

	return yaman.Attach(rootDir, args[0], opts)
}
