package yaman

import "github.com/willdurand/containers/internal/yaman/shim"

// Delete deletes a container.
func Delete(rootDir, id string) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	return shim.Destroy()
}
