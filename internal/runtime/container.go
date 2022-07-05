package runtime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/willdurand/containers/internal/constants"
)

type BaseContainer struct {
	Spec          runtimespec.Spec
	State         runtimespec.State
	CreatedAt     time.Time
	BaseDir       string
	StateFilePath string
}

func New(rootDir string, id string, bundleDir string) (*BaseContainer, error) {
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

	spec, err := LoadSpec(bundleDir)
	if bundleDir != "" && err != nil {
		return nil, err
	}

	if err := os.MkdirAll(containerDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	return &BaseContainer{
		Spec: spec,
		State: runtimespec.State{
			Version: runtimespec.Version,
			ID:      id,
			Status:  constants.StateCreating,
			Bundle:  bundleDir,
		},
		CreatedAt:     time.Now(),
		BaseDir:       containerDir,
		StateFilePath: filepath.Join(containerDir, "state.json"),
	}, nil
}

func Load(rootDir string, id string) (*BaseContainer, error) {
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

func LoadWithBundleConfig(rootDir string, id string) (*BaseContainer, error) {
	// Create a new container without bundle, which will create the container
	// state *without* the OCI bundle configuration. This is OK because we are
	// going to load the state right after, which will contain the path to the
	// bundle. From there, we'll be able to load the bundle config.
	container, err := Load(rootDir, id)
	if err != nil {
		return container, err
	}

	spec, err := LoadSpec(container.State.Bundle)
	if err != nil {
		return container, err
	}
	container.Spec = spec

	return container, nil
}

func (c *BaseContainer) ID() string {
	return c.State.ID
}

func (c *BaseContainer) IsCreated() bool {
	return c.State.Status == constants.StateCreated
}

func (c *BaseContainer) IsRunning() bool {
	return c.State.Status == constants.StateRunning
}

func (c *BaseContainer) IsStopped() bool {
	return c.State.Status == constants.StateStopped
}

func (c *BaseContainer) Rootfs() string {
	rootfs := c.Spec.Root.Path
	if !filepath.IsAbs(rootfs) {
		rootfs = filepath.Join(c.State.Bundle, rootfs)
	}
	return rootfs
}

func (c *BaseContainer) UpdateStatus(newStatus string) error {
	c.State.Status = newStatus
	return c.Save()
}

func (c *BaseContainer) Save() error {
	if err := c.saveContainerState(); err != nil {
		return err
	}

	return nil
}

func (c *BaseContainer) SetPid(pid int) {
	c.State.Pid = pid
}

func (c *BaseContainer) SaveAsCreated() error {
	return c.UpdateStatus(constants.StateCreated)
}

func (c *BaseContainer) Destroy() error {
	if err := os.RemoveAll(c.BaseDir); err != nil {
		return err
	}

	return nil
}

func (c *BaseContainer) loadContainerState() error {
	data, err := ioutil.ReadFile(c.StateFilePath)
	if err != nil {
		return fmt.Errorf("failed to read state.json: %w", err)
	}

	if err := json.Unmarshal(data, &c.State); err != nil {
		return fmt.Errorf("failed to parse state.json: %w", err)
	}

	return nil
}

func (c *BaseContainer) refreshContainerState() error {
	if c.State.Pid == 0 || c.IsStopped() {
		return nil
	}

	data, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/stat", c.State.Pid))
	// One character from the string "RSDZTW" where R is running, S is sleeping in an interruptible wait, D is waiting in uninterruptible disk sleep, Z is zombie, T is traced or stopped (on a signal), and W is paging.
	if err != nil || bytes.SplitN(data, []byte{' '}, 3)[2][0] == 'Z' {
		return c.UpdateStatus(constants.StateStopped)
	}

	return nil
}

func (c *BaseContainer) saveContainerState() error {
	data, err := json.Marshal(c.State)
	if err != nil {
		return fmt.Errorf("failed to serialize container state: %w", err)
	}

	if err := ioutil.WriteFile(c.StateFilePath, data, 0o644); err != nil {
		return fmt.Errorf("failed to save container state: %w", err)
	}

	return nil
}
