package container

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "logs <container>",
		Short:             "Fetch the logs of a container",
		Run:               cli.HandleErrors(logs),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	cmd.Flags().BoolP("timestamps", "t", false, "show timestamps")
	containerCommand.AddCommand(cmd)
}

func logs(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	timestamps, _ := cmd.Flags().GetBool("timestamps")

	opts := yaman.CopyLogsOpts{
		Timestamps: timestamps,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
	}

	return yaman.CopyLogs(rootDir, args[0], opts)
}
