package cmd

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/cmd/yacr/containers"
	"github.com/willdurand/containers/cmd/yacr/ipc"
	"github.com/willdurand/containers/internal/constants"
)

func init() {
	rootCmd.AddCommand(
		&cobra.Command{
			Use:          "start <id>",
			Short:        "Start a container",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				rootDir, _ := cmd.Flags().GetString("root")
				container, err := containers.LoadWithBundleConfig(rootDir, args[0])
				if err != nil {
					return fmt.Errorf("start: %w", err)
				}

				if !container.IsCreated() {
					return fmt.Errorf("start: unexpected status '%s' for container '%s'", container.State().Status, container.ID())
				}

				// Connect to the container.
				sockAddr, err := container.GetSockAddr(true)
				if err != nil {
					return fmt.Errorf("start: %w", err)
				}

				conn, err := net.Dial("unix", sockAddr)
				if err != nil {
					return fmt.Errorf("start: failed to dial container socket: %w", err)
				}
				defer conn.Close()

				// Hooks to be run before the container process is executed.
				// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#prestart
				if err := container.ExecuteHooks("Prestart"); err != nil {
					return fmt.Errorf("start: %w", err)
				}

				if err := ipc.SendMessage(conn, ipc.START_CONTAINER); err != nil {
					return fmt.Errorf("start: %w", err)
				}

				container.UpdateStatus(constants.StateRunning)

				// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#poststart
				if err := container.ExecuteHooks("Poststart"); err != nil {
					return fmt.Errorf("start: %w", err)
				}

				logrus.WithFields(logrus.Fields{
					"id": container.ID(),
				}).Info("start: (probably) ok")

				return nil
			},
			Args: cobra.ExactArgs(1),
		},
	)
}
