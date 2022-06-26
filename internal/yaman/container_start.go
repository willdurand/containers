package yaman

import (
	"github.com/willdurand/containers/internal/yaman/shim"
)

type StartOpts struct {
	Attach      bool
	Interactive bool
}

type StartResult struct {
	ExitStatus int
}

func Start(rootDir, id string, opts StartOpts) (StartResult, error) {
	var result StartResult

	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return result, err
	}

	attachDone := make(chan error)

	if opts.Attach || opts.Interactive {
		// Attach before starting the container to make sure we can receive all
		// the data when the container starts.
		go func() {
			attachDone <- shim.Attach(
				opts.Interactive && shim.Container.Opts.Interactive,
				true,
				true,
			)
		}()
	} else {
		close(attachDone)
	}

	if err := shim.StartContainer(); err != nil {
		return result, err
	}

	err = <-attachDone
	if err != nil {
		return result, err
	}

	state, err := shim.GetState()
	if err != nil {
		return result, err
	}

	result = StartResult{
		ExitStatus: state.Status.ExitStatus(),
	}

	return result, nil
}
