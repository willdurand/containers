package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
)

// ShimConfig represents the shim configuration.
type ShimConfig struct {
	rootDir         string
	debug           bool
	bundle          string
	containerId     string
	containerStatus *ContainerStatus
	runtime         string
	runtimePath     string
}

// NewShimConfigFromFlags creates a new shim configuration from a set of
// (command) flags. This function also verifies that required flags have
// non-empty values.
func NewShimConfigFromFlags(flags *pflag.FlagSet) (*ShimConfig, error) {
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

	return newShimConfig(rootDir, bundle, containerId, runtime, debug)
}

// newShimConfig creates a new shim configuration.
func newShimConfig(rootDir, bundle, containerId, runtime string, debug bool) (*ShimConfig, error) {
	containerDir := filepath.Join(rootDir, containerId)
	if err := os.MkdirAll(containerDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime '%s' not found", runtime)
	}

	return &ShimConfig{
		rootDir:         containerDir,
		debug:           debug,
		bundle:          bundle,
		containerId:     containerId,
		containerStatus: nil,
		runtime:         runtime,
		runtimePath:     runtimePath,
	}, nil
}

// Debug returns `true` when the debug mode is enabled on the shim, and `false`
// otherwise.
func (c *ShimConfig) Debug() bool {
	return c.debug
}

// Bundle returns the path to the container's bundle.
func (c *ShimConfig) Bundle() string {
	return c.bundle
}

// ContainerID returns the container's ID.
func (c *ShimConfig) ContainerID() string {
	return c.containerId
}

// ContainerStatus returns a pointer to the container status when it exists, and
// `nil` otherwise. A shim configuration should only have an instance of
// `ContainerStatus` when a container process has been created.
func (c *ShimConfig) ContainerStatus() *ContainerStatus {
	return c.containerStatus
}

// SetContainerStatus sets an instance of `ContainerStatus` to the shim
// configuration.
func (c *ShimConfig) SetContainerStatus(status *ContainerStatus) {
	c.containerStatus = status
}

// Runtime returns the name of the OCI runtime used by the shim.
func (c *ShimConfig) Runtime() string {
	return c.runtime
}

// RuntimePath returns the path to the OCI runtime binary used by the shim.
func (c *ShimConfig) RuntimePath() string {
	return c.runtimePath
}

// RuntimeArgs returns a list of common OCI runtime arguments.
func (c *ShimConfig) RuntimeArgs() []string {
	args := []string{
		"--log", filepath.Join(c.rootDir, fmt.Sprintf("%s.log", c.runtime)),
		"--log-format", "json",
	}
	if c.debug {
		args = append(args, "--debug")
	}

	return args
}

// ContainerPidFileName returns the path to the file that contains the PID of
// the container. Usually, this path should be passed to the OCI runtime with a
// CLI flag (`--pid-file`).
func (c *ShimConfig) ContainerPidFileName() string {
	return filepath.Join(c.rootDir, "container.pid")
}

// ShimPidFileName returns the path to the file that contains the PID of the
// shim.
func (c *ShimConfig) ShimPidFileName() string {
	return filepath.Join(c.rootDir, "shim.pid")
}

// SocketAddress returns the path to the unix socket used to communicate with
// the shim.
func (c *ShimConfig) SocketAddress() string {
	return filepath.Join(c.rootDir, "shim.sock")
}

// StdoutFileName is the path to the file where the container's `stdout` logs
// are written.
func (c *ShimConfig) StdoutFileName() string {
	return filepath.Join(c.rootDir, "stdout")
}

// StderrFileName is the path to the file where the container's `stderr` logs
// are written.
func (c *ShimConfig) StderrFileName() string {
	return filepath.Join(c.rootDir, "stderr")
}

// Destroy removes the directory (and all the files) created by the shim.
func (c *ShimConfig) Destroy() {
	os.RemoveAll(c.rootDir)
}
