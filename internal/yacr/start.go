package yacr

import (
	"fmt"
	"net"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/yacr/container"
	"github.com/willdurand/containers/internal/yacr/ipc"
)

func Start(rootDir, containerId string) error {

	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		return err
	}

	if !container.IsCreated() {
		return fmt.Errorf("start: unexpected status '%s' for container '%s'", container.State().Status, container.ID())
	}

	// Connect to the container.
	sockAddr, err := container.GetSockAddr(true)
	if err != nil {
		return err
	}

	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("start: failed to dial container socket: %w", err)
	}
	defer conn.Close()

	// Hooks to be run before the container process is executed.
	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#prestart
	if err := container.ExecuteHooks("Prestart"); err != nil {
		return err
	}

	if err := ipc.SendMessage(conn, ipc.START_CONTAINER); err != nil {
		return err
	}

	container.UpdateStatus(constants.StateRunning)

	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#poststart
	if err := container.ExecuteHooks("Poststart"); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Info("start: (probably) ok")
	return nil
}
