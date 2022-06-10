package cmd

import "github.com/willdurand/containers/pkg/cli"

const (
	programName string = "yacr"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = cli.NewRootCommand(
	programName,
	"Yet another (unsafe) container runtime",
)

func Execute() {
	cli.Execute(rootCmd)
}
