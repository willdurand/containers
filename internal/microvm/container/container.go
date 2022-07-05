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

func (c *MicrovmContainer) ArgsForQEMU(pidFile string, debug bool) []string {
	return []string{
		"-M", "microvm",
		"-m", "512m",
		"-no-acpi", "-no-reboot", "-no-user-config", "-nodefaults", "-display", "none",
		"-device", "virtio-serial-device",
		"-device", "virtconsole,chardev=virtiocon0",
		"-chardev", fmt.Sprintf("socket,id=virtiofs0,path=%s", c.VirtiofsdSocketPath()),
		"-device", "vhost-user-fs-device,queue-size=1024,chardev=virtiofs0,tag=/dev/root",
		// TODO: fixme
		"-kernel", "/workspace/containers/microvm/build/vmlinux",
		"-object", fmt.Sprintf("memory-backend-file,id=mem,size=%s,mem-path=%s,share=on", "512m", filepath.Join(c.BaseDir, "shm")),
		"-numa", "node,memdev=mem",
		"-pidfile", pidFile, "-daemonize",
		"-append", c.appendLine(debug),
	}
}

func (c *MicrovmContainer) appendLine(debug bool) string {
	args := []string{
		// Issue a keyboard controller reset to reboot. It's fine to reboot
		// because we pass `-no-reboot` to QEMU.
		"reboot=k",
		// We use virtio-fs for the root filesystem.
		"rootfstype=virtiofs", "root=/dev/root", "rw",
		// `hvc0` is the virtio-console configured when we start QEMU.
		"console=hvc0",
	}

	if debug {
		args = append(args, "MV_DEBUG=1")
	} else {
		args = append(args, "quiet", "MV_DEBUG=0")
	}

	args = append(
		args,
		fmt.Sprintf("MV_HOSTNAME=%s", c.Spec.Hostname),
		fmt.Sprintf("MV_INIT=%s", strings.Join(c.Spec.Process.Args, " ")),
	)

	return strings.Join(append(args, c.Spec.Process.Env...), " ")
}
