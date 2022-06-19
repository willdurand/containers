package yaman

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"os/exec"
	"strconv"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yaman/container"
)

func ProcessHook(r io.Reader, hookName string) error {
	var state runtimespec.State
	if err := json.NewDecoder(r).Decode(&state); err != nil {
		return err
	}

	logger := logrus.WithField("id", state.ID)

	switch hookName {
	case "CreateRuntime":
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

		logger.WithField("pid", slirp.Process.Pid).Debug("slirp4netns started")

		return slirp.Process.Release()
	}

	return nil
}
