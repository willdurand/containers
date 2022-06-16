package registry

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/willdurand/containers/internal/yaman/image"
)

// PullOpts contains options for the pull operation.
type PullOpts struct{}

// Pull downloads and unpacks an image from a registry.
func Pull(img *image.Image, opts PullOpts) error {
	_, err := os.Stat(img.BaseDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	switch img.Hostname {
	case "docker.io":
		if err := PullFromDocker(img); err != nil {
			return err
		}

	default:
		return fmt.Errorf("unsupported registry '%s'", img.Hostname)
	}

	return nil
}
