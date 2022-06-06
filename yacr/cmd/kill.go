package cmd

import (
	"fmt"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacr/containers"
)

func init() {
	cmd := &cobra.Command{
		Use:          "kill <id> <signal>",
		Short:        "Send a signal to a container",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			signal := 15 // SIGTERM

			if len(args) > 1 {
				sig, err := strconv.Atoi(args[1])
				if err != nil {
					return fmt.Errorf("kill: failed to parse signal value: %w", err)
				}
				signal = sig
			}

			rootDir, _ := cmd.Flags().GetString("root")
			container, err := containers.Load(rootDir, id)
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
