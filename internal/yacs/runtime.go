package yacs

import (
	"bytes"
	"encoding/json"
	"errors"
	"os/exec"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
)

const (
	runtimeLogFileName = "runtime.log"
)

var (
	ErrContainerNotExist = errors.New("container does not exist")
)

// State calls the OCI runtime and returns the runtime state or an error.
func (y *Yacs) State() (*runtimespec.State, error) {
	output, err := y.executeRuntime("state", y.containerID)
	if err != nil {
		return nil, err
	}

	state := new(runtimespec.State)
	if json.Unmarshal(output, state); err != nil {
		return nil, err
	}

	return state, nil
}

// Starts calls the OCI runtime to start the container.
func (y *Yacs) Start() error {
	_, err := y.executeRuntime("start", y.containerID)
	return err
}

// Kill calls the OCI runtime to send a signal to the container.
func (y *Yacs) Kill(signal string) error {
	_, err := y.executeRuntime("kill", y.containerID, signal)
	return err
}

// Sigkill calls the OCI runtime to send a `SIGKILL` signal to the container.
func (y *Yacs) Sigkill() error {
	return y.Kill("SIGKILL")
}

// Delete calls the OCI runtime to delete a container. It can be used to force
// dele the container as well.
func (y *Yacs) Delete(force bool) error {
	deleteArgs := []string{"delete", y.containerID}
	if force {
		deleteArgs = append(deleteArgs, "--force")
	}

	_, err := y.executeRuntime(deleteArgs...)
	return err
}

// executeRuntime calls the OCI runtime with the arguments passed to it.
func (y *Yacs) executeRuntime(args ...string) ([]byte, error) {
	c := exec.Command(y.runtimePath, append(y.runtimeArgs(), args...)...)
	logrus.WithField("command", c.String()).Debug("call OCI runtime")

	output, err := c.Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// HACK: we should probably not parse the error message like that...
			// Note that this should work with `runc` too, though.
			if bytes.Contains(exitError.Stderr, []byte("does not exist")) {
				return output, ErrContainerNotExist
			}
			// Adjust error with the stderr output instead of a generic message
			// like "exit status 1".
			err = errors.New(string(exitError.Stderr))
		}
	}

	return output, err
}

// runtimeArgs returns a list of common OCI runtime arguments.
func (y *Yacs) runtimeArgs() []string {
	args := []string{
		// We specify a log file so that the container's stderr is "clean" (because
		// the default log file is `/dev/stderr`).
		"--log", y.runtimeLogFilePath(),
		// We set the format to JSON because we might need to read this log file
		// in case of an error when creating the container.
		"--log-format", "json",
	}

	// Forward the debug state to the OCI runtime.
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		args = append(args, "--debug")
	}

	return args
}

func (y *Yacs) runtimeLogFilePath() string {
	return filepath.Join(y.baseDir, runtimeLogFileName)
}
