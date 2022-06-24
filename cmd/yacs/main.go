package main

import (
	"fmt"

	"github.com/sevlyar/go-daemon"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/yacs"
)

func main() {
	rootCmd := cli.NewRootCommand("yacs", "Yet another container shim")
	rootCmd.Run = cli.HandleErrors(run)
	rootCmd.Args = cobra.NoArgs

	rootCmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	rootCmd.MarkFlagRequired("bundle")
	rootCmd.Flags().String("container-id", "", "container id")
	rootCmd.MarkFlagRequired("container-id")
	rootCmd.Flags().String("container-log-file", "", `path to the container log file (default: "container.log")`)
	rootCmd.Flags().String("exit-command", "", "path to the exit command executed when the container has exited")
	rootCmd.Flags().StringArray("exit-command-arg", []string{}, "argument to pass to the execute command")
	rootCmd.Flags().String("runtime", "yacr", "container runtime to use")
	rootCmd.Flags().String("stdio-dir", "", "the directory to use when creating the stdio named pipes")

	cli.Execute(rootCmd)
}

func run(cmd *cobra.Command, args []string) error {
	// The code below (until `ctx.Reborn()`) is shared between a "parent" and a
	// "child" process. Both initialize Yacs but most of the logic lives in the
	// "child" process.

	shim, err := yacs.NewShimFromFlags(cmd.Flags())
	if err != nil {
		return err
	}

	ctx := &daemon.Context{
		PidFileName: shim.PidFilePath(),
		PidFilePerm: 0o644,
	}

	child, err := ctx.Reborn()
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}

	// This block is the "parent" process (in a fork/exec model). We wait until
	//receive a message from the "child" process.
	if child != nil {
		if err := shim.Err(); err != nil {
			return err
		}

		// When the shim has started successfully, we print the unix socket
		// address so that another program can interact with the shim.
		fmt.Println(shim.SocketPath())
		return nil
	}

	// This is the "child" process.
	defer ctx.Release()

	return shim.Run()
}
