package main

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:          "runtime",
		Short:        "Hidden command used to execute a container runtime",
		SilenceUsage: true,
		Hidden:       true,
		RunE:         runtime,
		Args:         cobra.NoArgs,
	}

	cmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	cmd.MarkFlagRequired("bundle")
	cmd.Flags().String("container-id", "", "container id")
	cmd.MarkFlagRequired("container-id")
	cmd.Flags().String("container-pidfile", "", "container pid file")
	cmd.MarkFlagRequired("container-pidfile")
	cmd.Flags().String("runtime", "", "container runtime to use")
	cmd.MarkFlagRequired("runtime")

	shimCmd.AddCommand(cmd)
}

func runtime(cmd *cobra.Command, args []string) error {
	containerId, _ := cmd.Flags().GetString("container-id")
	bundle, _ := cmd.Flags().GetString("bundle")
	pidFile, _ := cmd.Flags().GetString("container-pidfile")

	logger := logrus.WithFields(logrus.Fields{
		"id":  containerId,
		"cmd": "runtime",
	})

	runtime, _ := cmd.Flags().GetString("runtime")
	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return fmt.Errorf("runtime executable '%s' not found", runtime)
	}

	process := &exec.Cmd{
		Path: runtimePath,
		Args: appendGlobalFlags(
			cmd,
			[]string{
				runtime,
				"create", containerId,
				"--bundle", bundle,
				"--pid-file", pidFile,
			},
		),
		Stdin:  nil,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}

	logger.WithFields(logrus.Fields{
		"command": process.String(),
	}).Debug("executing container runtime")

	if err := process.Run(); err != nil {
		logger.WithError(err).Error("failed to execute container runtime")
		return err
	}

	logger.Debug("ok")
	return nil
}
