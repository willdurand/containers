package microvm

import (
	"fmt"
	"syscall"

	"github.com/willdurand/containers/internal/microvm/container"
)

func Kill(rootDir, containerId string, signal syscall.Signal) error {
	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return err
	}

	if !container.IsCreated() && !container.IsRunning() {
		return fmt.Errorf("unexpected status '%s' for container '%s'", container.State.Status, container.ID())
	}

	if container.State.Pid != 0 {
		if err := syscall.Kill(container.State.Pid, signal); err != nil {
			return fmt.Errorf("failed to send signal '%d' to container '%s': %w", signal, container.ID(), err)
		}
	}

	return nil
}
