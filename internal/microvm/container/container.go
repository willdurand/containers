package container

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/willdurand/containers/internal/runtime"
)

type MicrovmContainer struct {
	*runtime.BaseContainer
}

func New(rootDir string, id string, bundleDir string) (*MicrovmContainer, error) {
	base, err := runtime.New(rootDir, id, bundleDir)
	return &MicrovmContainer{base}, err
}

func LoadWithBundleConfig(rootDir string, id string) (*MicrovmContainer, error) {
	base, err := runtime.LoadWithBundleConfig(rootDir, id)
	return &MicrovmContainer{base}, err
}

func (c *MicrovmContainer) PipePath() string {
	return filepath.Join(c.BaseDir, "virtiocon0")
}

func (c *MicrovmContainer) PipePathIn() string {
	return fmt.Sprintf("%s.in", c.PipePath())
}

func (c *MicrovmContainer) PipePathOut() string {
	return fmt.Sprintf("%s.out", c.PipePath())
}

func (c *MicrovmContainer) InitDirPath() string {
	return filepath.Join(c.Rootfs(), "sbin")
}

func (c *MicrovmContainer) InitFilePath() string {
	return filepath.Join(c.InitDirPath(), "init")
}

func (c *MicrovmContainer) VirtiofsdSocketPath() string {
	return filepath.Join(c.BaseDir, "virtiofsd.sock")
}

func (c *MicrovmContainer) AppendLine(debug bool) string {
	debugStr := "0"
	if debug {
		debugStr = "1"
	}

	return strings.Join(append(
		[]string{
			"quiet", "reboot=t",
			"rootfstype=virtiofs", "root=/dev/root", "rw",
			"console=hvc0",
			fmt.Sprintf("MV_DEBUG=%s", debugStr),
			fmt.Sprintf("MV_HOSTNAME=%s", c.ID()),
			fmt.Sprintf("MV_INIT=%s", strings.Join(c.Spec.Process.Args, " ")),
		},
		c.Spec.Process.Env...,
	), " ")
}
