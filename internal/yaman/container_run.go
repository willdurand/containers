package yaman

import (
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/image"
	"github.com/willdurand/containers/internal/yaman/registry"
	"github.com/willdurand/containers/internal/yaman/shim"
)

// Run runs a command in a new container. We return the ID of the container on
// success and an error otherwise.
func Run(rootDir, imageName string, containerOpts container.ContainerOpts, shimOpts shim.ShimOpts) (string, error) {
	img, err := image.New(rootDir, imageName)
	if err != nil {
		return "", err
	}

	pullOpts := registry.PullOpts{
		Policy: registry.PullMissing,
	}
	if err := registry.Pull(img, pullOpts); err != nil {
		return "", err
	}

	container := container.New(rootDir, img, containerOpts)
	defer func() {
		if !container.IsStarted() {
			container.Destroy()
		}
	}()
	if err := container.MakeBundle(); err != nil {
		return "", err
	}

	shim := shim.New(container, shimOpts)
	if err := shim.Start(rootDir); err != nil {
		return "", err
	}

	if err := shim.StartContainer(); err != nil {
		return "", err
	}

	return container.ID, nil
}