package image

import "github.com/spf13/cobra"

var imageCommand = &cobra.Command{
	Use:     "image",
	Aliases: []string{"i"},
	Short:   "Manage images",
}

func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(imageCommand)
}
