package container

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
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
	Name        string
	Command     []string
	Remove      bool
	Hostname    string
	Interactive bool
	Tty         bool
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
	logFileName            = "container.log"
	slirp4netnsPidFileName = "slirp4netns.pid"
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

func GetBaseDir(rootDir string) string {
	return filepath.Join(rootDir, "containers")
}

func GetSlirp4netnsPidFilePath(bundleDir string) string {
	return filepath.Join(bundleDir, slirp4netnsPidFileName)
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
		"target": c.RootFS(),
		"fuse":   c.UseFuse,
	}).Debug("mount overlay")

	if c.UseFuse {
		if err := exec.Command(fuse, "-o", mountData, c.RootFS()).Run(); err != nil {
			return fmt.Errorf("failed to mount overlay (fuse): %w", err)
		}
	} else {
		if err := syscall.Mount("overlay", c.RootFS(), "overlay", 0, mountData); err != nil {
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

	self, err := os.Executable()
	if err != nil {
		return err
	}

	c.Config.Hooks = &runtimespec.Hooks{
		CreateRuntime: []runtimespec.Hook{
			runtimespec.Hook{
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

func (c *Container) CleanUp() error {
	if _, err := os.Stat(c.Slirp4netnsPidFilePath()); err == nil {
		if data, err := os.ReadFile(c.Slirp4netnsPidFilePath()); err == nil {
			if slirpPid, err := strconv.Atoi(string(bytes.TrimSpace(data))); err == nil {
				logrus.WithField("pid", slirpPid).Debug("terminating slirp4netns")

				if err := syscall.Kill(slirpPid, syscall.SIGTERM); err != nil {
					logrus.WithError(err).Debug("failed to terminate slirp4netns")
				}
			}
		}
	}

	if c.UseFuse {
		if err := exec.Command("fusermount3", "-u", c.RootFS()).Run(); err != nil {
			logrus.WithError(err).Debug("failed to unmount rootfs (fuse)")
		}
	} else {
		if err := syscall.Unmount(c.RootFS(), 0); err != nil {
			// This likely happens because the rootfs has been previously unmounted.
			logrus.WithError(err).Debug("failed to unmount rootfs (native)")
		}
	}

	return nil
}

func (c *Container) Destroy() error {
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

func (c *Container) Slirp4netnsPidFilePath() string {
	return GetSlirp4netnsPidFilePath(c.BaseDir)
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
