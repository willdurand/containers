package container

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
)

func init() {
	cmd := &cobra.Command{
		Use:               "start <container>",
		Short:             "Start a container",
		Run:               cli.HandleErrors(start),
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeContainerIds,
	}
	cmd.Flags().BoolP("attach", "a", false, "attach stdio streams")
	cmd.Flags().BoolP("interactive", "i", false, "keep stdin open")
	containerCommand.AddCommand(cmd)
}

func start(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	attach, _ := cmd.Flags().GetBool("attach")
	interactive, _ := cmd.Flags().GetBool("interactive")

	opts := yaman.StartOpts{
		Attach:      attach,
		Interactive: interactive,
	}

	result, err := yaman.Start(rootDir, args[0], opts)
	if err != nil {
		return err
	}

	os.Exit(result.ExitStatus)
	return nil
}
