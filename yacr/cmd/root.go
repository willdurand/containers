package cmd

import (
	"github.com/willdurand/containers/cmd"
)

const (
	programName string = "yacr"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = cmd.NewRootCommand(
	programName,
	"Yet another (unsafe) container runtime",
)

func Execute() {
	cmd.Execute(rootCmd)
}
