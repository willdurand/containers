package yaman

import (
	"io"

	"github.com/willdurand/containers/internal/yaman/shim"
)

// CopyLogsOpts contains the options for the `CopyLogs` function.
type CopyLogsOpts struct {
	Timestamps bool
	Stdout     io.Writer
	Stderr     io.Writer
}

// CopyLogs copies the logs of a container to the writers specified in the
// options.
func CopyLogs(rootDir, id string, opts CopyLogsOpts) error {
	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return err
	}

	return shim.CopyLogs(opts.Stdout, opts.Stderr, opts.Timestamps)
}
