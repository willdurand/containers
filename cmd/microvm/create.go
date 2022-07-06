package main

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm"
)

func init() {
	createCmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")
			bundle, _ := cmd.Flags().GetString("bundle")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			debug, _ := cmd.Flags().GetBool("debug")

			opts := microvm.CreateOpts{
				PidFile:       pidFile,
				ConsoleSocket: consoleSocket,
				Debug:         debug,
			}

			return microvm.Create(rootDir, args[0], bundle, opts)
		}),
		Args: cobra.ExactArgs(1),
	}
	createCmd.PersistentFlags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	createCmd.MarkFlagRequired("bundle")
	createCmd.Flags().String("pid-file", "", "specify the file to write the process id to")
	createCmd.Flags().String("console-socket", "", "console unix socket used to pass a PTY descriptor")
	rootCmd.AddCommand(createCmd)
}
