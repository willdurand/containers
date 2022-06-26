package yaman

import (
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/registry"
	"github.com/willdurand/containers/internal/yaman/shim"
)

// RunResult represents the return value of the `Run` function.
type RunResult struct {
	ContainerID string
	ExitStatus  int
}

// Run runs a command in a new container. We return the ID of the container on
// success and an error otherwise.
func Run(rootDir, imageName string, pullOpts registry.PullOpts, containerOpts container.ContainerOpts, shimOpts shim.ShimOpts) (RunResult, error) {
	var result RunResult

	_, container, err := Create(rootDir, imageName, pullOpts, containerOpts, shimOpts)
	if err != nil {
		return result, err
	}

	startOpts := StartOpts{
		Attach:      !containerOpts.Detach,
		Interactive: containerOpts.Interactive,
	}

	sr, err := Start(rootDir, container.ID, startOpts)
	if err != nil {
		return result, err
	}

	result = RunResult{
		ContainerID: container.ID,
		ExitStatus:  sr.ExitStatus,
	}

	return result, nil
}
