package container

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
	"github.com/willdurand/containers/internal/yaman/registry"
	"github.com/willdurand/containers/internal/yaman/shim"
)

func init() {
	cmd := &cobra.Command{
		Use:   "run <image> [<command> [<args>...]]",
		Short: "Run a command in a new container",
		Run:   cli.HandleErrors(run),
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.Flags().BoolP("detach", "d", false, "run container in background and print container ID")
	addCreateFlagsToCommand(cmd)
	containerCommand.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	// registry/pull options
	pull, _ := cmd.Flags().GetString("pull")
	pullPolicy, err := registry.ParsePullPolicy(pull)
	if err != nil {
		return err
	}
	pullOpts := registry.PullOpts{
		Policy: pullPolicy,
		Output: os.Stderr,
	}

	// container options
	containerOpts := makeContainerOptsFromCommand(cmd, args[1:])
	detach, _ := cmd.Flags().GetBool("detach")
	if detach {
		containerOpts.Detach = true
	}

	// shim options
	shimOpts := shim.ShimOpts{}
	if runtime, _ := cmd.Flags().GetString("runtime"); runtime != "" {
		shimOpts.Runtime = runtime
	}

	result, err := yaman.Run(rootDir, args[0], pullOpts, containerOpts, shimOpts)
	if err != nil {
		// If we do not have an `ExitCodeError` already, set the exit code to
		// `126` to indicate a problem coming from Yaman.
		switch err.(type) {
		case cli.ExitCodeError:
			return err
		default:
			return cli.ExitCodeError{Message: err.Error(), ExitCode: 126}
		}
	}

	// In detached mode, we print the container ID to the standard output and we
	// are done. The container should be running as long as it is supposed to
	// (e.g., if the command exits after completion, the container might be
	// exited but if the command is a daemon, the container should still be
	// alive).
	if detach {
		fmt.Fprintln(os.Stdout, result.ContainerID)
		return nil
	}

	os.Exit(result.ExitStatus)
	return nil
}
