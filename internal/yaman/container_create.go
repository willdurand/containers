package yaman

import (
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/image"
	"github.com/willdurand/containers/internal/yaman/registry"
	"github.com/willdurand/containers/internal/yaman/shim"
)

func Create(rootDir, imageName string, pullOpts registry.PullOpts, containerOpts container.ContainerOpts, shimOpts shim.ShimOpts) (*shim.Shim, *container.Container, error) {
	img, err := image.New(rootDir, imageName)
	if err != nil {
		return nil, nil, err
	}

	if err := registry.Pull(img, pullOpts); err != nil {
		return nil, nil, err
	}

	container, err := container.New(rootDir, img, containerOpts)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if !container.IsCreated() {
			container.Delete()
		}
	}()

	shim := shim.New(container, shimOpts)
	if err := shim.Create(rootDir); err != nil {
		return nil, nil, err
	}

	return shim, container, nil
}
