package main

import (
	"fmt"
	"syscall"

	"github.com/docker/docker/pkg/signal"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yacr/container"
)

func init() {
	cmd := &cobra.Command{
		Use:          "kill <id> <signal>",
		Short:        "Send a signal to a container",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			signal, err := signal.ParseSignal(args[1])
			if err != nil {
				signal = syscall.SIGTERM
			}

			rootDir, _ := cmd.Flags().GetString("root")
			container, err := container.LoadWithBundleConfig(rootDir, id)
			if err != nil {
				return fmt.Errorf("kill: %w", err)
			}

			if !container.IsCreated() && !container.IsRunning() {
				return fmt.Errorf("kill: unexpected status '%s' for container '%s'", container.State().Status, container.ID())
			}

			if container.State().Pid != 0 {
				if err := syscall.Kill(container.State().Pid, syscall.Signal(signal)); err != nil {
					return fmt.Errorf("kill: failed to send signal '%d' to container '%s': %w", signal, container.ID(), err)
				}
			}

			logrus.WithFields(logrus.Fields{
				"id":     container.ID(),
				"signal": signal,
			}).Info("kill: ok")

			return nil
		},
		Args: cobra.MinimumNArgs(1),
	}
	cmd.Flags().Bool("all", false, "UNSUPPORTED FLAG")

	rootCmd.AddCommand(cmd)
}
