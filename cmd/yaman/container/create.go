package container

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yaman"
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/registry"
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
	cmd.Flags().String("entrypoint", "", "overwrite the default entrypoint set by the image")
	cmd.Flags().String("hostname", "", "set the container hostname")
	cmd.Flags().BoolP("interactive", "i", false, "keep stdin open")
	cmd.Flags().BoolP("publish-all", "P", false, "publish all exposed ports to random ports")
	cmd.Flags().String("pull", string(registry.PullMissing), `pull image before running ("always"|"missing"|"never")`)
	cmd.Flags().Bool("rm", false, "automatically remove the container when it exits")
	cmd.Flags().String("runtime", "", "runtime to use for this container")
	cmd.Flags().BoolP("tty", "t", false, "allocate a pseudo-tty")
}

func makeContainerOptsFromCommand(cmd *cobra.Command, command []string) container.ContainerOpts {
	var entrypoint []string
	entrypointStr, _ := cmd.Flags().GetString("entrypoint")
	if entrypointStr != "" {
		if err := json.Unmarshal([]byte(entrypointStr), &entrypoint); err != nil {
			logrus.WithError(err).Debug("failed to parse entrypoint as JSON")
			entrypoint = []string{entrypointStr}
		}
	}

	hostname, _ := cmd.Flags().GetString("hostname")
	interactive, _ := cmd.Flags().GetBool("interactive")
	publishAll, _ := cmd.Flags().GetBool("publish-all")
	rm, _ := cmd.Flags().GetBool("rm")
	tty, _ := cmd.Flags().GetBool("tty")

	return container.ContainerOpts{
		Command:     command,
		Entrypoint:  entrypoint,
		Remove:      rm,
		Hostname:    hostname,
		Interactive: interactive,
		Tty:         tty,
		Detach:      false,
		PublishAll:  publishAll,
	}
}

func create(cmd *cobra.Command, args []string) error {
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

	// shim options
	shimOpts := shim.ShimOpts{}
	if runtime, _ := cmd.Flags().GetString("runtime"); runtime != "" {
		shimOpts.Runtime = runtime
	}

	_, container, err := yaman.Create(rootDir, args[0], pullOpts, containerOpts, shimOpts)
	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, container.ID)
	return nil
}
