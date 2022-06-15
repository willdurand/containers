package image

import "github.com/spf13/cobra"

var imageCommand = &cobra.Command{
	Use:   "image",
	Short: "Manage images",
}

func Register(rootCmd *cobra.Command) {
	rootCmd.AddCommand(imageCommand)
}
