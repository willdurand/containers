package yaman

import "github.com/willdurand/containers/internal/yaman/shim"

func CleanUp(rootDir, id string) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	if err := shim.Terminate(); err != nil {
		return err
	}

	if shim.Container.Opts.Remove {
		return shim.Destroy()
	}

	return nil
}
