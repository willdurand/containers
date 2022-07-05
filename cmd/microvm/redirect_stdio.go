package main

import (
	"io"
	"os"
	"sync"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm/container"
)

func init() {
	redirectStdioCmd := &cobra.Command{
		Use:    "redirect-stdio <id>",
		Short:  "redirect the microvm standard IOs",
		Hidden: true,
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")

			container, err := container.LoadWithBundleConfig(rootDir, args[0])
			if err != nil {
				return err
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()

				file, err := os.OpenFile(container.PipePathOut(), os.O_RDONLY, 0o600)
				if err != nil {
					logrus.WithError(err).Error("open: pipe out")
					return
				}
				defer file.Close()

				if _, err := io.Copy(os.Stdout, file); err != nil {
					logrus.WithError(err).Error("copy: pipe out")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()

				file, err := os.OpenFile(container.PipePathIn(), os.O_WRONLY, 0o600)
				if err != nil {
					logrus.WithError(err).Error("open: pipe in")
					return
				}
				defer file.Close()

				if _, err := io.Copy(file, os.Stdin); err != nil {
					logrus.WithError(err).Error("copy: pipe in")
				}
			}()

			wg.Wait()

			return nil
		}),
	}
	rootCmd.AddCommand(redirectStdioCmd)
}
