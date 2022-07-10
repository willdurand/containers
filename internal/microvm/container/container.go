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

// kernelPath is the path to the kernel binary on the host. This path should be
// kept in sync with the `make -C microvm install_kernel` command.
var kernelPath = "/usr/lib/microvm/vmlinux"

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
	return filepath.Join(c.BaseDir, "vfsd.sock")
}

func (c *MicrovmContainer) ArgsForQEMU(pidFile string, debug, tty bool) []string {
	return []string{
		"-M", "microvm",
		"-m", "512m",
		"-no-acpi", "-no-reboot", "-no-user-config", "-nodefaults", "-display", "none",
		"-device", "virtio-serial-device",
		"-device", "virtconsole,chardev=virtiocon0",
		"-chardev", fmt.Sprintf("socket,id=virtiofs0,path=%s", c.VirtiofsdSocketPath()),
		"-device", "vhost-user-fs-device,queue-size=1024,chardev=virtiofs0,tag=/dev/root",
		"-kernel", kernelPath,
		"-object", fmt.Sprintf("memory-backend-memfd,id=mem,size=%s,share=on", "512m"),
		"-numa", "node,memdev=mem",
		"-pidfile", pidFile, "-daemonize",
		"-append", c.appendLine(debug, tty),
	}
}

func (c *MicrovmContainer) appendLine(debug, tty bool) string {
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

	if tty {
		args = append(args, "MV_TTY=1")
	} else {
		args = append(args, "MV_TTY=0")
	}

	args = append(
		args,
		fmt.Sprintf("MV_HOSTNAME=%s", c.Spec.Hostname),
		fmt.Sprintf("MV_INIT=%s", strings.Join(c.Spec.Process.Args, " ")),
	)

	return strings.Join(append(args, c.Spec.Process.Env...), " ")
}
