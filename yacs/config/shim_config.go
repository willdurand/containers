package config

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/pflag"
)

type ShimConfig struct {
	rootDir         string
	debug           bool
	bundle          string
	containerId     string
	containerStatus *ContainerStatus
	runtime         string
	runtimePath     string
}

func NewShimConfigFromFlags(flags *pflag.FlagSet) (*ShimConfig, error) {
	for _, param := range []string{
		"bundle",
		"container-id",
		"runtime",
	} {
		if v, err := flags.GetString(param); v == "" || err != nil {
			return nil, fmt.Errorf("invalid value for '--%s'", param)
		}
	}

	rootDir, _ := flags.GetString("root")
	bundle, _ := flags.GetString("bundle")
	containerId, _ := flags.GetString("container-id")
	runtime, _ := flags.GetString("runtime")
	debug, _ := flags.GetBool("debug")

	return NewShimConfig(rootDir, bundle, containerId, runtime, debug)
}

func NewShimConfig(rootDir, bundle, containerId, runtime string, debug bool) (*ShimConfig, error) {
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

func (c *ShimConfig) Debug() bool {
	return c.debug
}

func (c *ShimConfig) Bundle() string {
	return c.bundle
}

func (c *ShimConfig) ContainerID() string {
	return c.containerId
}

func (c *ShimConfig) ContainerStatus() *ContainerStatus {
	return c.containerStatus
}

func (c *ShimConfig) SetContainerStatus(status *ContainerStatus) {
	c.containerStatus = status
}

func (c *ShimConfig) Runtime() string {
	return c.runtime
}

func (c *ShimConfig) RuntimePath() string {
	return c.runtimePath
}

func (c *ShimConfig) ContainerPidFileName() string {
	return filepath.Join(c.rootDir, "container.pid")
}

func (c *ShimConfig) ShimPidFileName() string {
	return filepath.Join(c.rootDir, "shim.pid")
}

func (c *ShimConfig) SocketAddress() string {
	return filepath.Join(c.rootDir, "shim.sock")
}

func (c *ShimConfig) StdoutFileName() string {
	return filepath.Join(c.rootDir, "stdout")
}

func (c *ShimConfig) StderrFileName() string {
	return filepath.Join(c.rootDir, "stderr")
}

func (c *ShimConfig) Destroy() {
	os.RemoveAll(c.rootDir)
}
