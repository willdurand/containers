package yaman

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strconv"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yaman/container"
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
		slirp4netns, err := exec.LookPath("slirp4netns")
		if err != nil {
			return err
		}

		slirp := exec.Command(slirp4netns, []string{
			"--configure",
			"--mtu=65520",
			"--disable-host-loopback",
			strconv.Itoa(state.Pid),
			"en0",
		}...)

		logger.WithField("command", slirp.String()).Debug("starting slirp4netns")

		if err := slirp.Start(); err != nil {
			return err
		}

		if err := ioutil.WriteFile(
			container.GetSlirp4netnsPidFilePath(state.Bundle),
			[]byte(strconv.FormatInt(int64(slirp.Process.Pid), 10)),
			0o644,
		); err != nil {
			return err
		}

		// Configure DNS inside the container.
		if err := ioutil.WriteFile(
			filepath.Join(shim.Container.RootFS(), "etc", "resolv.conf"),
			[]byte("nameserver 10.0.2.3\n"),
			0o644,
		); err != nil {
			return err
		}

		logger.WithField("pid", slirp.Process.Pid).Debug("slirp4netns started")

		return slirp.Process.Release()
	}

	return nil
}
