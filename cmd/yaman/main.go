// Yet Another (Container) MANager.
package main

import (
	"github.com/willdurand/containers/cmd/yaman/container"
	"github.com/willdurand/containers/cmd/yaman/image"
	"github.com/willdurand/containers/internal/cli"
)

const (
	programName string = "yaman"
)

// rootCmd represents the root command.
var rootCmd = cli.NewRootCommand(
	programName,
	"Yet another daemonless container manager",
)

func init() {
	image.Register(rootCmd)
	container.Register(rootCmd)
}

func main() {
	cli.Execute(rootCmd)
}
