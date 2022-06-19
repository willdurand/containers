package container

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/runtime"
	"github.com/willdurand/containers/internal/yacr/ipc"
)

type BaseContainer struct {
	spec          runtimespec.Spec
	state         runtimespec.State
	createdAt     time.Time
	rootDir       string
	stateFilePath string
}

type ContainerState struct {
	BaseContainer
}

func New(rootDir string, id string, bundleDir string) (*ContainerState, error) {
	containerDir := filepath.Join(rootDir, id)

	if bundleDir != "" {
		if _, err := os.Stat(containerDir); err == nil {
			return nil, fmt.Errorf("container '%s' already exists", id)
		}

		if !filepath.IsAbs(bundleDir) {
			absoluteDir, err := filepath.Abs(bundleDir)
			if err != nil {
				return nil, err
			}
			bundleDir = absoluteDir
		}
	}

	spec, err := runtime.LoadSpec(bundleDir)
	if bundleDir != "" && err != nil {
		return nil, err
	}

	baseContainer := BaseContainer{
		spec: spec,
		state: runtimespec.State{
			Version: runtimespec.Version,
			ID:      id,
			Status:  constants.StateCreating,
			Bundle:  bundleDir,
		},
		createdAt:     time.Now(),
		rootDir:       containerDir,
		stateFilePath: filepath.Join(containerDir, "state.json"),
	}

	return &ContainerState{baseContainer}, nil
}

func Load(rootDir string, id string) (*ContainerState, error) {
	container, err := New(rootDir, id, "")
	if err != nil {
		return container, err
	}

	if err := container.loadContainerState(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return container, fmt.Errorf("container '%s' does not exist", id)
		}

		return container, err
	}

	if err := container.refreshContainerState(); err != nil {
		return container, err
	}

	return container, nil
}

func LoadWithBundleConfig(rootDir string, id string) (*ContainerState, error) {
	// Create a new container without bundle, which will create the container
	// state *without* the OCI bundle configuration. This is OK because we are
	// going to load the state right after, which will contain the path to the
	// bundle. From there, we'll be able to load the bundle config.
	container, err := Load(rootDir, id)
	if err != nil {
		return container, err
	}

	spec, err := runtime.LoadSpec(container.state.Bundle)
	if err != nil {
		return container, err
	}
	container.spec = spec

	return container, nil
}

func LoadFromContainer(rootDir string, id string) (*BaseContainer, error) {
	container := &BaseContainer{}

	c, err := LoadWithBundleConfig(rootDir, id)
	if err != nil {
		return container, err
	}

	container.spec = c.spec
	container.state = c.state
	// See: https://github.com/opencontainers/runtime-spec/blob/a3c33d663ebc56c4d35dbceaa447c7bf37f6fab3/runtime.md#state
	container.state.Pid = os.Getpid()
	container.createdAt = c.createdAt
	container.rootDir = c.rootDir
	container.stateFilePath = ""

	return container, nil
}

func (c *BaseContainer) ID() string {
	return c.state.ID
}

func (c *BaseContainer) Spec() runtimespec.Spec {
	return c.spec
}

func (c *BaseContainer) State() runtimespec.State {
	return c.state
}

func (c *BaseContainer) CreatedAt() time.Time {
	return c.createdAt
}

func (c *BaseContainer) IsCreated() bool {
	return c.State().Status == constants.StateCreated
}

func (c *BaseContainer) IsRunning() bool {
	return c.State().Status == constants.StateRunning
}

func (c *BaseContainer) IsStopped() bool {
	return c.State().Status == constants.StateStopped
}

func (c *BaseContainer) Rootfs() string {
	rootfs := c.Spec().Root.Path
	if !filepath.IsAbs(rootfs) {
		rootfs = filepath.Join(c.State().Bundle, rootfs)
	}
	return rootfs
}

func (c *BaseContainer) GetInitSockAddr(mustExist bool) (string, error) {
	initSockAddr := filepath.Join(c.rootDir, "init.sock")
	return initSockAddr, ipc.EnsureValidSockAddr(initSockAddr, mustExist)
}

func (c *BaseContainer) GetSockAddr(mustExist bool) (string, error) {
	sockAddr := filepath.Join(c.rootDir, "ipc.sock")
	return sockAddr, ipc.EnsureValidSockAddr(sockAddr, mustExist)
}

func (c *BaseContainer) ExecuteHooks(name string) error {
	if c.spec.Hooks == nil {
		return nil
	}

	hooks := map[string][]runtimespec.Hook{
		"Prestart":        c.spec.Hooks.Prestart,
		"CreateRuntime":   c.spec.Hooks.CreateRuntime,
		"CreateContainer": c.spec.Hooks.CreateContainer,
		"StartContainer":  c.spec.Hooks.StartContainer,
		"Poststart":       c.spec.Hooks.Poststart,
		"Poststop":        c.spec.Hooks.Poststop,
	}[name]

	if len(hooks) == 0 {
		logrus.WithFields(logrus.Fields{
			"id":    c.ID(),
			"name:": name,
		}).Debug("no hooks")

		return nil
	}

	logrus.WithFields(logrus.Fields{
		"id":    c.ID(),
		"name:": name,
		"state": c.state,
		"hooks": hooks,
	}).Debug("executing hooks")

	s, err := json.Marshal(c.state)
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		var stdout, stderr bytes.Buffer

		cmd := exec.Cmd{
			Path:   hook.Path,
			Args:   hook.Args,
			Env:    hook.Env,
			Stdin:  bytes.NewReader(s),
			Stdout: &stdout,
			Stderr: &stderr,
		}

		if err := cmd.Run(); err != nil {
			logrus.WithFields(logrus.Fields{
				"id":     c.ID(),
				"name:":  name,
				"error":  err,
				"stderr": stderr.String(),
				"stdout": stdout.String(),
			}).Error("failed to execute hooks")

			return fmt.Errorf("failed to execute %s hook '%s': %w", name, cmd.String(), err)
		}
	}

	return nil
}

func (c *ContainerState) UpdateStatus(newStatus string) error {
	c.state.Status = newStatus
	return c.Save()
}

func (c *ContainerState) Save() error {
	if err := os.MkdirAll(c.rootDir, 0o755); err != nil {
		return fmt.Errorf("failed to create container directory: %w", err)
	}

	if err := c.saveContainerState(); err != nil {
		return err
	}

	return nil
}

func (c *ContainerState) SetPid(pid int) {
	c.state.Pid = pid
}

func (c *ContainerState) SaveAsCreated() error {
	return c.UpdateStatus(constants.StateCreated)
}

func (c *ContainerState) Destroy() error {
	if err := os.RemoveAll(c.rootDir); err != nil {
		return err
	}

	return nil
}

func (c *ContainerState) loadContainerState() error {
	data, err := ioutil.ReadFile(c.stateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read state.json: %w", err)
	}

	if err := json.Unmarshal(data, &c.state); err != nil {
		return fmt.Errorf("failed to parse state.json: %w", err)
	}

	return nil
}

func (c *ContainerState) refreshContainerState() error {
	if c.State().Pid == 0 || c.IsStopped() {
		return nil
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", c.State().Pid))
	// One character from the string "RSDZTW" where R is running, S is sleeping in an interruptible wait, D is waiting in uninterruptible disk sleep, Z is zombie, T is traced or stopped (on a signal), and W is paging.
	if err != nil || bytes.SplitN(data, []byte{' '}, 3)[2][0] == 'Z' {
		return c.UpdateStatus(constants.StateStopped)
	}

	return nil
}

func (c *ContainerState) saveContainerState() error {
	data, err := json.Marshal(c.state)
	if err != nil {
		return fmt.Errorf("failed to serialize container state: %w", err)
	}

	if err := ioutil.WriteFile(c.stateFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	return nil
}
