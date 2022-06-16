package container

import (
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman"
)

var containerCommand = &cobra.Command{
	Use:     "container",
	Aliases: []string{"c"},
	Short:   "Manage containers",
}

func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(containerCommand)
}

func completeContainerIds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) != 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	return completeManyContainerIds(cmd, args, toComplete)
}

func completeManyContainerIds(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	rootDir, _ := cmd.Flags().GetString("root")

	return yaman.GetContainerIds(rootDir, toComplete), cobra.ShellCompDirectiveNoFileComp
}
