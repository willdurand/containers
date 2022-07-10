package main

import "github.com/willdurand/containers/internal/cli"

const (
	programName string = "microvm"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = cli.NewRootCommand(
	programName,
	"An experimental runtime backed by micro VMs",
)

func init() {
	rootCmd.PersistentFlags().Bool("systemd-cgroup", false, "UNSUPPORTED FLAG")
}

func main() {
	cli.Execute(rootCmd)
}
