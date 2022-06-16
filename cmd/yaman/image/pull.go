package image

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/yaman/image"
	"github.com/willdurand/containers/internal/yaman/registry"
)

func init() {
	cmd := &cobra.Command{
		Use:          "pull <image>",
		Short:        "Pull an image from a registry",
		RunE:         pull,
		SilenceUsage: true,
		Args:         cobra.ExactArgs(1),
	}
	imageCommand.AddCommand(cmd)
}

func pull(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")

	img, err := image.New(rootDir, args[0])
	if err != nil {
		return err
	}

	opts := registry.PullOpts{
		Policy: registry.PullAlways,
	}
	if err := registry.Pull(img, opts); err != nil {
		return err
	}

	fmt.Printf("downloaded %s\n", img.FQIN())

	return nil
}
