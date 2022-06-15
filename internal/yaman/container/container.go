package container

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/runtime"
	"github.com/willdurand/containers/internal/yaman/image"
)

type ContainerOpts struct {
	Name     string
	Detach   bool
	Command  []string
	Remove   bool
	Hostname string
	Tty      bool
}

type Container struct {
	ID        string
	BaseDir   string
	Image     *image.Image
	Config    *runtimespec.Spec
	Opts      ContainerOpts
	CreatedAt time.Time
	StartedAt time.Time
	ExitedAt  time.Time
	LogFile   string
}

const containerLogFileName = "container.log"

func New(rootDir string, img *image.Image, opts ContainerOpts) *Container {
	id := strings.ReplaceAll(uuid.NewString(), "-", "")
	baseDir := filepath.Join(GetBaseDir(rootDir), id)

	return &Container{
		ID:        id,
		BaseDir:   baseDir,
		Image:     img,
		Opts:      opts,
		CreatedAt: time.Now(),
		LogFile:   filepath.Join(baseDir, containerLogFileName),
	}
}

func GetBaseDir(rootDir string) string {
	return filepath.Join(rootDir, "containers")
}

func (c *Container) RootFS() string {
	return filepath.Join(c.BaseDir, "rootfs")
}

func (c *Container) MakeBundle() error {
	imageConfig, err := c.Image.Config()
	if err != nil {
		return err
	}

	for _, dir := range []string{
		c.BaseDir,
		c.datadir(),
		c.workdir(),
		c.RootFS(),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}

	mountData := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", c.lowerdir(), c.datadir(), c.workdir())

	logrus.WithFields(logrus.Fields{
		"data":   mountData,
		"target": c.RootFS(),
	}).Debug("mount overlay")

	if err := syscall.Mount("overlay", c.RootFS(), "overlay", 0, mountData); err != nil {
		return fmt.Errorf("failed to mount overlay: %w", err)
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

	c.Config = runtime.BaseSpec(c.RootFS())
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

	data, err := json.Marshal(c.Config)
	if err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(c.BaseDir, "config.json"), data, 0o644); err != nil {
		return err
	}

	return nil
}

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

func (c *Container) Cleanup() error {
	if err := syscall.Unmount(c.RootFS(), 0); err != nil {
		// This likely happens because the rootfs has been previously unmounted.
		logrus.WithError(err).Debug("failed to unmount rootfs")

	}

	return nil
}

func (c *Container) Destroy() error {
	if err := c.Cleanup(); err != nil {
		return err
	}

	if err := os.RemoveAll(c.BaseDir); err != nil {
		return err
	}

	logrus.WithField("id", c.ID).Debug("container destroyed")
	return nil
}

func (c *Container) IsStarted() bool {
	return !c.StartedAt.IsZero()
}

func (c *Container) IsExited() bool {
	return !c.ExitedAt.IsZero()
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
