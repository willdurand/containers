package yacr

import (
	"fmt"
	"syscall"

	"github.com/docker/docker/pkg/signal"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yacr/container"
)

func Kill(rootDir string, args []string) error {
	containerId := args[0]

	sig := syscall.SIGTERM
	if len(args) > 1 {
		if s, err := signal.ParseSignal(args[1]); err == nil {
			sig = s
		}
	}

	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	if !container.IsCreated() && !container.IsRunning() {
		return fmt.Errorf("unexpected status '%s' for container '%s'", container.State().Status, container.ID())
	}

	if container.State().Pid != 0 {
		if err := syscall.Kill(container.State().Pid, syscall.Signal(sig)); err != nil {
			return fmt.Errorf("failed to send signal '%d' to container '%s': %w", sig, container.ID(), err)
		}
	}

	logrus.WithFields(logrus.Fields{
		"id":     container.ID(),
		"signal": sig,
	}).Info("ok")

	return nil
}
