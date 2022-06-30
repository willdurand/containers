package yaman

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yaman/network"
	"github.com/willdurand/containers/internal/yaman/shim"
)

func ProcessHook(rootDir, hookName string, r io.Reader) error {
	var state runtimespec.State
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	shim, err := shim.Load(rootDir, state.ID)
	if err != nil {
		return err
	}

	logger := logrus.WithField("id", state.ID)

	switch hookName {
	case "network-setup":
		slirp4netns, err := network.NewSlirp4netns(state.Pid, shim.Slirp4netnsApiSocketPath())
		if err != nil {
			return err
		}

		pid, err := slirp4netns.Start()
		if err != nil {
			return err
		}

		// Write PID file for later. Note that we could have used the exit-fd as
		// well since this PID file is mainly used to terminate the slirp4netns
		// process when we clean-up the container.
		if err := ioutil.WriteFile(
			shim.Slirp4netnsPidFilePath(),
			[]byte(strconv.Itoa(pid)),
			0o644,
		); err != nil {
			return err
		}

		// Configure DNS inside the container.
		if err := ioutil.WriteFile(
			filepath.Join(shim.Container.Rootfs(), "etc", "resolv.conf"),
			[]byte("nameserver 10.0.2.3\n"),
			0o644,
		); err != nil {
			logger.WithError(err).Warn("failed to write /etc/resolv.conf")
		}

		// Expose ports
		if len(shim.Container.ExposedPorts) > 0 {
			// TODO: use ready-FD instead...
			time.Sleep(50 * time.Millisecond)
			if err := slirp4netns.ExposePorts(shim.Container.ExposedPorts); err != nil {
				return err
			}
		}
	}

	return nil
}
