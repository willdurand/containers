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
	cmd.Flags().String("hostname", "", "set the container hostname")
	cmd.Flags().BoolP("interactive", "i", false, "keep stdin open")
	cmd.Flags().String("name", "", "assign a name to the container")
	cmd.Flags().Bool("rm", false, "automatically remove the container when it exits")
	cmd.Flags().String("runtime", "", "runtime to use for this container")
	cmd.Flags().BoolP("tty", "t", false, "allocate a pseudo-tty")
	containerCommand.AddCommand(cmd)
}

func run(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	detach, _ := cmd.Flags().GetBool("detach")

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
	containerOpts := container.ContainerOpts{
		Name:        name,
		Command:     args[1:],
		Remove:      rm,
		Hostname:    hostname,
		Interactive: interactive,
		Tty:         tty,
	}

	// shim options
	shimOpts := shim.ShimOpts{}
	if runtime, _ := cmd.Flags().GetString("runtime"); runtime != "" {
		shimOpts.Runtime = runtime
	}

	containerId, err := yaman.Run(rootDir, args[0], containerOpts, shimOpts)
	if err != nil {
		return err
	}

	// In detached mode, we print the container ID to the standard output and we
	// are done. The container should be running as long as it is supposed to
	// (e.g., if the command exits after completion, the container might be
	// exited but if the command is a daemon, the container should still be
	// alive).
	if detach {
		fmt.Fprintln(os.Stdout, containerId)
		return nil
	}

	attachOpts := yaman.AttachOpts{
		In:  os.Stdin,
		Out: os.Stdout,
		Err: os.Stderr,
	}

	return yaman.Attach(rootDir, containerId, attachOpts)
}
