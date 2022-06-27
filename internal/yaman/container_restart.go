package yaman

import (
	"github.com/willdurand/containers/internal/yaman/shim"
)

func Restart(rootDir, id string) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	if err := shim.Recreate(rootDir); err != nil {
		return err
	}

	sOpts := StartOpts{
		Attach:      false,
		Interactive: false,
	}

	_, err = Start(rootDir, id, sOpts)
	return err
}
