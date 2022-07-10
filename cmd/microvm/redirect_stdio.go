package main

import (
	"bufio"
	"fmt"
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
			debug, _ := cmd.Flags().GetBool("debug")

			container, err := container.LoadWithBundleConfig(rootDir, args[0])
			if err != nil {
				return err
			}

			// So... We need to wait until the VM is "ready" to send STDIN data,
			// otherwise STDIN might be ECHO'ed on STDOUT. I am not too sure why this
			// happens (maybe that's how the Linux console is configured?) so I
			// introduced a workaround...
			//
			// The `init(1)` process will disable ECHO and print a special message
			// for us here. When we receive it, we can copy the data.
			vmReady := make(chan interface{})

			pipeOut, err := os.OpenFile(container.PipePathOut(), os.O_RDONLY, 0o600)
			if err != nil {
				return err
			}
			defer pipeOut.Close()

			s := bufio.NewScanner(pipeOut)
			for s.Scan() {
				if debug {
					fmt.Println(s.Text())
				}
				// Should be kept in sync with `microvm/init.c`.
				if s.Text() == "init: ready" {
					break
				}
			}

			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()

				close(vmReady)

				if _, err := io.Copy(os.Stdout, pipeOut); err != nil {
					logrus.WithError(err).Error("copy: pipe out")
				}
			}()

			wg.Add(1)
			go func() {
				defer wg.Done()

				pipeIn, err := os.OpenFile(container.PipePathIn(), os.O_WRONLY, 0o600)
				if err != nil {
					logrus.WithError(err).Error("open: pipe in")
					return
				}
				defer pipeIn.Close()

				<-vmReady

				if _, err := io.Copy(pipeIn, os.Stdin); err != nil {
					logrus.WithError(err).Error("copy: pipe in")
				}
			}()

			wg.Wait()

			return nil
		}),
	}
	rootCmd.AddCommand(redirectStdioCmd)
}
