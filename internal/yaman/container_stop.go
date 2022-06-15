package yaman

import "github.com/willdurand/containers/internal/yaman/shim"

func Stop(rootDir, id string) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	return shim.StopContainer()
}
