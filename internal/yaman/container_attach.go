package yaman

import (
	"os"

	"github.com/willdurand/containers/internal/yaman/shim"
)

type AttachOpts struct {
	In  *os.File
	Out *os.File
	Err *os.File
}

func Attach(rootDir, id string, opts AttachOpts) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	return shim.Attach(opts.In, opts.Out, opts.Err)
}
