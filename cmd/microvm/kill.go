package main

import (
	"syscall"

	"github.com/docker/docker/pkg/signal"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm"
)

func init() {
	killCmd := &cobra.Command{
		Use:   "kill <id> [<signal>]",
		Short: "Send a signal to a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			sig := syscall.SIGTERM
			if len(args) > 1 {
				if s, err := signal.ParseSignal(args[1]); err == nil {
					sig = s
				}
			}

			return microvm.Kill(rootDir, args[0], sig)
		}),
		Args: cobra.MinimumNArgs(1),
	}
	killCmd.Flags().Bool("all", false, "UNSUPPORTED FLAG")
	rootCmd.AddCommand(killCmd)
}
