package container

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/cmd"
	"github.com/willdurand/containers/internal/runtime"
	"github.com/willdurand/containers/internal/yaman/image"
)

type ContainerOpts struct {
	Name        string
	Command     []string
	Remove      bool
	Hostname    string
	Interactive bool
	Tty         bool
	Detach      bool
}

type Container struct {
	ID          string
	BaseDir     string
	Image       *image.Image
	Config      *runtimespec.Spec
	Opts        ContainerOpts
	CreatedAt   time.Time
	StartedAt   time.Time
	ExitedAt    time.Time
	LogFilePath string
	UseFuse     bool
}

const (
	logFileName = "container.log"
)

func New(rootDir string, img *image.Image, opts ContainerOpts) *Container {
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	baseDir := filepath.Join(GetBaseDir(rootDir), id)

	return &Container{
		ID:          id,
		BaseDir:     baseDir,
		Image:       img,
		Opts:        opts,
		CreatedAt:   time.Now(),
		LogFilePath: filepath.Join(baseDir, logFileName),
	}
}

// Rootfs returns the absolute path to the root filesystem.
func (c *Container) Rootfs() string {
	return filepath.Join(c.BaseDir, "rootfs")
}

// Command returns the container's command, which is what gets executed in the
// container when it starts.
func (c *Container) Command() []string {
	var args []string
	if conf, err := c.Image.Config(); err == nil {
		args = conf.Config.Entrypoint
		if len(c.Opts.Command) > 0 {
			args = append(args, c.Opts.Command...)
		} else {
			args = append(args, conf.Config.Cmd...)
		}
	}

	return args
}

// Mount creates a bundle configuration for the container and mounts its root
// filesystem.
func (c *Container) Mount() error {
	imageConfig, err := c.Image.Config()
	if err != nil {
		return err
	}

	for _, dir := range []string{
		c.BaseDir,
		c.datadir(),
		c.workdir(),
		c.Rootfs(),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	mountData := fmt.Sprintf(
		"lowerdir=%s,upperdir=%s,workdir=%s",
		c.lowerdir(),
		c.datadir(),
		c.workdir(),
	)

	fuse, err := exec.LookPath("fuse-overlayfs")
	// We need `fuse-overlayfs` if we want to use it but when Yaman is executed
	// with elevated privileges, we can safely use the native OverlayFS.
	c.UseFuse = err == nil && os.Getuid() != 0

	logrus.WithFields(logrus.Fields{
		"data":   mountData,
		"target": c.Rootfs(),
		"fuse":   c.UseFuse,
	}).Debug("mount overlay")

	if c.UseFuse {
		if err := cmd.Run(exec.Command(fuse, "-o", mountData, c.Rootfs())); err != nil {
			return fmt.Errorf("failed to mount overlay (fuse): %w", err)
		}
	} else {
		if err := syscall.Mount("overlay", c.Rootfs(), "overlay", 0, mountData); err != nil {
			return fmt.Errorf("failed to mount overlay (native): %w", err)
		}
	}

	// Convert image config into a runtime config.
	// See: https://github.com/opencontainers/image-spec/blob/main/conversion.md
	cwd := "/"
	if imageConfig.Config.WorkingDir != "" {
		cwd = imageConfig.Config.WorkingDir
	}

	hostname := c.Opts.Hostname
	if hostname == "" {
		hostname = c.ID
	}

	c.Config, err = runtime.BaseSpec(c.Rootfs(), os.Getuid() != 0)
	if err != nil {
		return err
	}

	c.Config.Process = &runtimespec.Process{
		Terminal: c.Opts.Tty,
		User: runtimespec.User{
			UID: 0,
			GID: 0,
		},
		Args: c.Command(),
		Env:  imageConfig.Config.Env,
		Cwd:  cwd,
	}
	c.Config.Hostname = hostname

	self, err := os.Executable()
	if err != nil {
		return err
	}
	c.Config.Hooks = &runtimespec.Hooks{
		CreateRuntime: []runtimespec.Hook{
			{
				Path: self,
				Args: []string{self, "container", "hook", "network-setup"},
			},
		},
	}

	data, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(c.BaseDir, "config.json"), data, 0o644); err != nil {
		return err
	}

	return nil
}

// Unmount unmounts the root filesystem of the container.
func (c *Container) Unmount() error {
	if c.UseFuse {
		if err := cmd.Run(exec.Command("fusermount3", "-u", c.Rootfs())); err != nil {
			logrus.WithError(err).Debug("failed to unmount rootfs (fuse)")
		}
	} else {
		if err := syscall.Unmount(c.Rootfs(), 0); err != nil {
			// This likely happens because the rootfs has been previously unmounted.
			logrus.WithError(err).Debug("failed to unmount rootfs (native)")
		}
	}

	return nil
}

// IsStarted returns `true` when the container has started, and `false` otherwise.
func (c *Container) IsStarted() bool {
	return !c.StartedAt.IsZero()
}

// IsExited returns `true` when the container has exited, and `false` otherwise.
func (c *Container) IsExited() bool {
	return !c.ExitedAt.IsZero()
}

// Delete removes the container base directory and all its files.
func (c *Container) Delete() error {
	if err := os.RemoveAll(c.BaseDir); err != nil {
		return err
	}

	logrus.WithField("id", c.ID).Debug("container deleted")
	return nil
}

func (c *Container) lowerdir() string {
	return strings.Join(c.Image.LayerDirs(), ":")
}

func (c *Container) datadir() string {
	return filepath.Join(c.BaseDir, "data")
}

func (c *Container) workdir() string {
	return filepath.Join(c.BaseDir, "work")
}
