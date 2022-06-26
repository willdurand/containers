package yaman

import (
	"github.com/willdurand/containers/internal/yaman/shim"
)

type AttachOpts struct {
	Stdin  bool
	Stdout bool
	Stderr bool
}

func Attach(rootDir, id string, opts AttachOpts) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	return shim.Attach(opts.Stdin, opts.Stdout, opts.Stderr)
}
