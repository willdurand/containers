package shim

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
)

type Shim struct {
	rootDir         string
	debug           bool
	bundle          string
	containerId     string
	containerStatus *ContainerStatus
	runtime         string
	runtimePath     string
	exitCommand     string
	exitCommandArgs []string
	Exit            chan struct{}
}

// NewFromFlags creates a new shim configuration from a set of (command) flags.
// This function also verifies that required flags have non-empty values.
func NewFromFlags(flags *pflag.FlagSet) (*Shim, error) {
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
	runtime, _ := flags.GetString("runtime")
	debug, _ := flags.GetBool("debug")
	exitCommand, _ := flags.GetString("exit-command")
	exitCommandArgs, _ := flags.GetStringArray("exit-command-arg")

	return newShimConfig(rootDir, bundle, containerId, runtime, exitCommand, exitCommandArgs, debug)
}

// newShimConfig creates a new shim configuration.
func newShimConfig(rootDir, bundle, containerId, runtime string, exitProgram string, exitCommandArgs []string, debug bool) (*Shim, error) {
	containerDir := filepath.Join(rootDir, containerId)
	if err := os.MkdirAll(containerDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime '%s' not found", runtime)
	}
	return &Shim{
		rootDir:         containerDir,
		debug:           debug,
		bundle:          bundle,
		containerId:     containerId,
		containerStatus: nil,
		runtime:         runtime,
		runtimePath:     runtimePath,
		exitCommand:     exitProgram,
		exitCommandArgs: exitCommandArgs,
		Exit:            make(chan struct{}),
	}, nil
}

// ContainerID returns the container's ID.
func (s *Shim) ContainerID() string {
	return s.containerId
}

// PidFileName returns the path to the file that contains the PID of the shim.
func (s *Shim) PidFileName() string {
	return filepath.Join(s.rootDir, "shim.pid")
}

// SocketAddress returns the path to the unix socket used to communicate with
// the shim.
func (s *Shim) SocketAddress() string {
	return filepath.Join(s.rootDir, "shim.sock")
}

// Destroy removes the directory (and all the files) created by the shim.
func (s *Shim) Destroy() {
	os.RemoveAll(s.rootDir)
}

// setContainerStatus sets an instance of `ContainerStatus` to the shim
// configuration.
func (s *Shim) setContainerStatus(status *ContainerStatus) {
	s.containerStatus = status
}

// runtimeArgs returns a list of common OCI runtime arguments.
func (s *Shim) runtimeArgs() []string {
	args := []string{
		"--log", filepath.Join(s.rootDir, fmt.Sprintf("%s.log", s.runtime)),
		"--log-format", "json",
	}
	if s.debug {
		args = append(args, "--debug")
	}

	return args
}

// stdoutFileName is the path to the file where the container's `stdout` logs
// are written.
func (s *Shim) stdoutFileName() string {
	return filepath.Join(s.rootDir, "stdout")
}

// stderrFileName is the path to the file where the container's `stderr` logs
// are written.
func (s *Shim) stderrFileName() string {
	return filepath.Join(s.rootDir, "stderr")
}

// containerPidFileName returns the path to the file that contains the PID of
// the container. Usually, this path should be passed to the OCI runtime with a
// CLI flag (`--pid-file`).
func (s *Shim) containerPidFileName() string {
	return filepath.Join(s.rootDir, "container.pid")
}
