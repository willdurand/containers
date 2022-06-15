package container

import "github.com/spf13/cobra"

var containerCommand = &cobra.Command{
	Use:     "container",
	Aliases: []string{"c"},
	Short:   "Manage containers",
}

func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(containerCommand)
}
