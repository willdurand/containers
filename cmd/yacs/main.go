package main

import (
	"fmt"

	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/cmd/yacs/yacs"
	"github.com/willdurand/containers/pkg/cmd"
	"golang.org/x/sys/unix"
)

const (
	programName string = "yacs"
)

// shimCmd represents the shim command (which is the base command).
var shimCmd = cmd.NewRootCommand(
	programName,
	"Yet another container shim",
)

func init() {
	// We want to execute a function by default.
	shimCmd.RunE = run
	shimCmd.Args = cobra.NoArgs

	shimCmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	shimCmd.MarkFlagRequired("bundle")
	shimCmd.Flags().String("container-id", "", "container id")
	shimCmd.MarkFlagRequired("container-id")
	shimCmd.Flags().String("runtime", "yacr", "container runtime to use")
}

func main() {
	cmd.Execute(shimCmd)
}

func run(cmd *cobra.Command, args []string) error {
	shim, err := yacs.NewFromFlags(cmd.Flags())
	if err != nil {
		return err
	}

	ctx := &daemon.Context{
		PidFileName: shim.PidFileName(),
		PidFilePerm: 0o644,
	}

	parent, err := ctx.Reborn()
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	if parent != nil {
		fmt.Println(shim.SocketAddress())
		return nil
	}
	defer ctx.Release()

	logger := logrus.WithFields(logrus.Fields{
		"id":  shim.ContainerID(),
		"cmd": "shim",
	})

	// The daemon shim has started. We cannot log information to stdout/stderr
	// so we are going to use `logger.Fatal()` in case of an error.
	logger.Info("started")

	// Make this daemon a subreaper so that it "adopts" orphaned descendants,
	// see: https://man7.org/linux/man-pages/man2/prctl.2.html
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		logger.WithError(err).Fatal("prctl() failed")
	}

	// Call the OCI runtime to create the container.
	go shim.CreateContainer(logger)

	// Create the HTTP API to be able to interact with the shim.
	go shim.CreateHttpServer(logger)

	<-shim.Exit

	shim.Destroy()
	logger.Info("stopped")

	return nil
}
