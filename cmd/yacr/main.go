// Yet Another Container Runtime
package main

import "github.com/willdurand/containers/internal/cli"

const (
	programName string = "yacr"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = cli.NewRootCommand(
	programName,
	"Yet another (unsafe) container runtime",
)

func init() {
	rootCmd.PersistentFlags().Bool("systemd-cgroup", false, "UNSUPPORTED FLAG")
}

func main() {
	cli.Execute(rootCmd)
}
