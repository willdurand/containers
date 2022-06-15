package yacs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
)

const (
	containerLogFileName = "container.log"
	containerPidFileName = "container.pid"
	runtimeLogFileName   = "runtime.log"
	shimPidFileName      = "shim.pid"
	shimSocketName       = "shim.sock"
)

type Yacs struct {
	baseDir          string
	bundleDir        string
	ContainerID      string
	ContainerLogFile string
	containerStatus  *ContainerStatus
	runtime          string
	runtimePath      string
	exitCommand      string
	exitCommandArgs  []string
	Exit             chan struct{}
}

// NewShimFromFlags creates a new shim from a set of (command) flags. This
// function also verifies that required flags have non-empty valuey.
func NewShimFromFlags(flags *pflag.FlagSet) (*Yacs, error) {
	for _, param := range []string{
		"bundle",
		"container-id",
		"runtime",
	} {
		if v, err := flags.GetString(param); err != nil || v == "" {
			return nil, fmt.Errorf("missing or invalid value for '--%s'", param)
		}
	}

	rootDir, _ := flags.GetString("root")

	bundle, _ := flags.GetString("bundle")
	containerId, _ := flags.GetString("container-id")
	containerLogFile, _ := flags.GetString("container-log-file")
	exitCommand, _ := flags.GetString("exit-command")
	exitCommandArgs, _ := flags.GetStringArray("exit-command-arg")
	runtime, _ := flags.GetString("runtime")

	baseDir := filepath.Join(rootDir, containerId)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime '%s' not found", runtime)
	}

	if containerLogFile == "" {
		containerLogFile = filepath.Join(baseDir, containerLogFileName)
	}

	return &Yacs{
		ContainerID:      containerId,
		ContainerLogFile: containerLogFile,
		Exit:             make(chan struct{}),
		baseDir:          baseDir,
		bundleDir:        bundle,
		containerStatus:  nil,
		runtime:          runtime,
		runtimePath:      runtimePath,
		exitCommand:      exitCommand,
		exitCommandArgs:  exitCommandArgs,
	}, nil
}

// PidFilePath returns the path to the file that contains the PID of the shim.
func (y *Yacs) PidFilePath() string {
	return filepath.Join(y.baseDir, shimPidFileName)
}

// SocketPath returns the path to the unix socket used to communicate with the
// shim.
func (y *Yacs) SocketPath() string {
	return filepath.Join(y.baseDir, shimSocketName)
}

// Destroy removes the directory (and all the files) created by the shim.
func (y *Yacs) Destroy() {
	if err := os.RemoveAll(y.baseDir); err != nil {
		logrus.WithError(err).Warn("failed to remove base directory")
	}
}

// setContainerStatus sets an instance of `ContainerStatus` to the shim
// configuration.
func (y *Yacs) setContainerStatus(status *ContainerStatus) {
	y.containerStatus = status
}

// runtimeArgs returns a list of common OCI runtime arguments.
func (y *Yacs) runtimeArgs() []string {
	args := []string{
		// We specify a log file so that the container's stderr is "clean" (because
		// the default log file is `/dev/stderr`).
		"--log", filepath.Join(y.baseDir, runtimeLogFileName),
	}

	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		args = append(args, "--debug")
	}

	return args
}

// containerPidFilePath returns the path to the file that contains the PID of
// the container. Usually, this path should be passed to the OCI runtime with a
// CLI flag (`--pid-file`).
func (y *Yacs) containerPidFilePath() string {
	return filepath.Join(y.baseDir, containerPidFileName)
}
