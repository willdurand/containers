package registry

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"

	"github.com/willdurand/containers/internal/yaman/image"
)

type PullPolicy string

// PullOpts contains options for the pull operation.
type PullOpts struct {
	Policy PullPolicy
	Output io.Writer
}

const (
	// PullAlways means that we always pull the image.
	PullAlways PullPolicy = "always"
	// PullMissing means that we pull the image if it does not already exist.
	PullMissing PullPolicy = "missing"
	// PullNever means that we never pull the image.
	PullNever PullPolicy = "never"
)

var ErrInvalidPullPolicy = errors.New("invalid pull policy")

// Pull downloads and unpacks an image from a registry.
func Pull(img *image.Image, opts PullOpts) error {
	if opts.Policy == PullNever {
		return nil
	}

	st, err := os.Stat(img.BaseDir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}

	if st != nil && st.IsDir() {
		if opts.Policy == PullMissing {
			return nil
		}

		if err := os.RemoveAll(img.BaseDir); err != nil {
			return err
		}
	}

	var rOpts registryOpts
	switch img.Hostname {
	case "docker.io":
		rOpts = registryOpts{
			AuthURL:      "https://auth.docker.io/token",
			Service:      "registry.docker.io",
			IndexBaseURL: "https://index.docker.io/v2",
		}

	case "quay.io":
		rOpts = registryOpts{
			AuthURL:      "https://quay.io/v2/auth",
			Service:      "quay.io",
			IndexBaseURL: "https://quay.io/v2",
		}

	default:
		return fmt.Errorf("unsupported registry '%s'", img.Hostname)
	}

	return PullFromRegistry(img, opts, rOpts)
}

func ParsePullPolicy(value string) (PullPolicy, error) {
	switch strings.ToLower(value) {
	case "always":
		return PullAlways, nil
	case "missing":
		return PullMissing, nil
	case "never":
		return PullNever, nil
	}

	return "", ErrInvalidPullPolicy
}
