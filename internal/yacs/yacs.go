package yacs

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
)

type Yacs struct {
	baseDir          string
	debug            bool
	bundle           string
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
	runtime, _ := flags.GetString("runtime")
	debug, _ := flags.GetBool("debug")
	exitCommand, _ := flags.GetString("exit-command")
	exitCommandArgs, _ := flags.GetStringArray("exit-command-arg")

	baseDir := filepath.Join(rootDir, containerId)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime '%s' not found", runtime)
	}

	if containerLogFile == "" {
		containerLogFile = filepath.Join(baseDir, "container.log")
	}

	return &Yacs{
		ContainerID:      containerId,
		ContainerLogFile: containerLogFile,
		Exit:             make(chan struct{}),
		baseDir:          baseDir,
		debug:            debug,
		bundle:           bundle,
		containerStatus:  nil,
		runtime:          runtime,
		runtimePath:      runtimePath,
		exitCommand:      exitCommand,
		exitCommandArgs:  exitCommandArgs,
	}, nil
}

// PidFileName returns the path to the file that contains the PID of the shim.
func (y *Yacs) PidFileName() string {
	return filepath.Join(y.baseDir, "shim.pid")
}

// SocketAddress returns the path to the unix socket used to communicate with
// the shim.
func (y *Yacs) SocketAddress() string {
	return filepath.Join(y.baseDir, "shim.sock")
}

// Destroy removes the directory (and all the files) created by the shim.
func (y *Yacs) Destroy() {
	os.RemoveAll(y.baseDir)
}

// setContainerStatus sets an instance of `ContainerStatus` to the shim
// configuration.
func (y *Yacs) setContainerStatus(status *ContainerStatus) {
	y.containerStatus = status
}

// runtimeArgs returns a list of common OCI runtime argumenty.
func (y *Yacs) runtimeArgs() []string {
	args := []string{
		"--log", filepath.Join(y.baseDir, fmt.Sprintf("%s.log", y.runtime)),
		"--log-format", "json",
	}
	if y.debug {
		args = append(args, "--debug")
	}

	return args
}

// containerPidFileName returns the path to the file that contains the PID of
// the container. Usually, this path should be passed to the OCI runtime with a
// CLI flag (`--pid-file`).
func (y *Yacs) containerPidFileName() string {
	return filepath.Join(y.baseDir, "container.pid")
}
