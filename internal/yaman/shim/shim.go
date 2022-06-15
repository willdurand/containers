package shim

import (
	"path/filepath"

	"github.com/willdurand/containers/internal/yaman/container"
)

// BaseShim is the base structure for a shim.
type BaseShim struct {
	Container *container.Container
	Opts      ShimOpts
}

// ShimOpts contains the options that can be passed to a shim.
type ShimOpts struct {
	Runtime string
}

const stateFileName = "shim.json"

// ID returns the ID of the shim, which is also the container's ID given that a
// shim is bound to a container (or the other way around) and container IDs are
// unique.
func (s *BaseShim) ID() string {
	return s.Container.ID
}

func (s *BaseShim) stateFilePath() string {
	return filepath.Join(s.Container.BaseDir, stateFileName)
}
