package yaman

import (
	"time"

	imagespec "github.com/opencontainers/image-spec/specs-go/v1"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/willdurand/containers/internal/yacs"
	"github.com/willdurand/containers/internal/yaman/container"
	"github.com/willdurand/containers/internal/yaman/image"
	"github.com/willdurand/containers/internal/yaman/shim"
)

// ContainerInspect is a data transfer structure and represents the result of
// the `inspect` command.
type ContainerInspect struct {
	Id      string
	Root    string
	Config  runtimespec.Spec
	Options container.ContainerOpts
	Created time.Time
	Started time.Time
	Exited  time.Time
	Image   struct {
		image.Image
		Config   imagespec.Image
		Manifest imagespec.Manifest
	}
	Shim struct {
		yacs.YacsState
		Options    shim.ShimOpts
		SocketPath string
	}
}

// Inspect returns low-level information about a container.
func Inspect(rootDir, id string) (ContainerInspect, error) {
	var inspect ContainerInspect

	shim, err := shim.Load(rootDir, id)
	if err != nil {
		return inspect, err
	}

	inspect.Id = shim.Container.ID
	inspect.Root = shim.Container.BaseDir
	inspect.Config = *shim.Container.Config
	inspect.Options = shim.Container.Opts
	inspect.Created = shim.Container.CreatedAt
	inspect.Started = shim.Container.StartedAt
	inspect.Exited = shim.Container.ExitedAt
	inspect.Image.Image = *shim.Container.Image
	inspect.Image.Config = *shim.Container.Image.Config
	inspect.Image.Manifest = *shim.Container.Image.Manifest
	if state, err := shim.GetState(); err == nil {
		inspect.Shim.YacsState = *state
	}
	inspect.Shim.Options = shim.Opts
	inspect.Shim.SocketPath = shim.SocketPath

	return inspect, nil
}
