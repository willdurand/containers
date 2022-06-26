package container

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
	"github.com/willdurand/containers/internal/yaman/container"
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
	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		rand.Seed(time.Now().UnixNano())
		name = namesgenerator.GetRandomName(0)
	}
	rm, _ := cmd.Flags().GetBool("rm")
	hostname, _ := cmd.Flags().GetString("hostname")
	interactive, _ := cmd.Flags().GetBool("interactive")
	tty, _ := cmd.Flags().GetBool("tty")
	detach, _ := cmd.Flags().GetBool("detach")
	containerOpts := container.ContainerOpts{
		Name:        name,
		Command:     args[1:],
		Remove:      rm,
		Hostname:    hostname,
		Interactive: interactive,
		Tty:         tty,
		Detach:      detach,
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
