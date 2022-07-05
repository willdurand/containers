package microvm

import (
	"fmt"

	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/microvm/container"
)

func Start(rootDir, containerId string) error {
	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return err
	}

	if !container.IsCreated() {
		return fmt.Errorf("start: unexpected status '%s' for container '%s'", container.State.Status, container.ID())
	}

	return container.UpdateStatus(constants.StateRunning)
}
