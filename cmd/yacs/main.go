package main

import (
	"fmt"

	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yacs"
	"golang.org/x/sys/unix"
)

const (
	programName string = "yacs"
)

// shimCmd represents the shim command (which is the base command).
var shimCmd = cli.NewRootCommand(
	programName,
	"Yet another container shim",
)

func init() {
	// We want to execute a function by default.
	shimCmd.Run = cli.HandleErrors(run)
	shimCmd.Args = cobra.NoArgs

	shimCmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	shimCmd.MarkFlagRequired("bundle")
	shimCmd.Flags().String("container-id", "", "container id")
	shimCmd.MarkFlagRequired("container-id")
	shimCmd.Flags().String(
		"container-log-file",
		"",
		`path to the container log file (default: "container.log" in the container base directory)`,
	)
	shimCmd.Flags().String("exit-command", "", "path to the exit command to execute when the container has exited")
	shimCmd.Flags().StringArray("exit-command-arg", []string{}, "argument to pass to the execute command")
	shimCmd.Flags().String("runtime", "yacr", "container runtime to use")
	shimCmd.Flags().String("stdio-dir", "", "the directory to use when creating the stdio named pipes")
}

func main() {
	cli.Execute(shimCmd)
}

func run(cmd *cobra.Command, args []string) error {
	shim, err := yacs.NewShimFromFlags(cmd.Flags())
	if err != nil {
		return err
	}

	ctx := &daemon.Context{
		PidFileName: shim.PidFilePath(),
		PidFilePerm: 0o644,
	}

	parent, err := ctx.Reborn()
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	if parent != nil {
		fmt.Println(shim.SocketPath())
		return nil
	}
	defer ctx.Release()

	logger := logrus.WithField("id", shim.ContainerID)

	// The daemon shim has started. We cannot log information to stdout/stderr so
	// we are going to use `logger.Panic()` in case of an error.
	logger.Info("daemon started")

	// Make this daemon a subreaper so that it "adopts" orphaned descendants,
	// see: https://man7.org/linux/man-pages/man2/prctl.2.html
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		logger.WithError(err).Panic("prctl() failed")
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
