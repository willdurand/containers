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
		Use:   "create <image> [<command> [<args>...]]",
		Short: "Create a new container",
		Run:   cli.HandleErrors(create),
		Args:  cobra.MinimumNArgs(1),
	}
	addCreateFlagsToCommand(cmd)
	containerCommand.AddCommand(cmd)
}

func addCreateFlagsToCommand(cmd *cobra.Command) {
	cmd.Flags().String("hostname", "", "set the container hostname")
	cmd.Flags().BoolP("interactive", "i", false, "keep stdin open")
	cmd.Flags().String("name", "", "assign a name to the container")
	cmd.Flags().Bool("rm", false, "automatically remove the container when it exits")
	cmd.Flags().String("runtime", "", "runtime to use for this container")
	cmd.Flags().BoolP("tty", "t", false, "allocate a pseudo-tty")
}

func create(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

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
		Detach:      true,
	}

	// shim options
	shimOpts := shim.ShimOpts{}
	if runtime, _ := cmd.Flags().GetString("runtime"); runtime != "" {
		shimOpts.Runtime = runtime
	}

	_, container, err := yaman.Create(rootDir, args[0], containerOpts, shimOpts)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, container.ID)
	return nil
}
