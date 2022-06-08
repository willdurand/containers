package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/cmd/yacr/containers"
	"github.com/willdurand/containers/cmd/yacr/ipc"
	"golang.org/x/sys/unix"
)

func container(cmd *cobra.Command, args []string) error {
	rootDir, _ := cmd.Flags().GetString("root")
	container, err := containers.LoadFromContainer(rootDir, args[0])
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}

	initSockAddr, err := container.GetInitSockAddr(true)
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id":           container.ID(),
		"initSockAddr": initSockAddr,
	}).Debug("container: starting")

	// Connect to the initial socket to tell the host (runtime) that this process has started.
	initConn, err := net.Dial("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("container: failed to dial init socket: %w", err)
	}
	defer initConn.Close()

	// Create a new socket to allow communication with this container.
	sockAddr, err := container.GetSockAddr(false)
	if err != nil {
		return fmt.Errorf("container: %w", err)
	}
	listener, err := net.Listen("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("container: listen error: %w", err)
	}
	defer listener.Close()

	// Notify the host that we are alive.
	if err := ipc.SendMessage(initConn, ipc.CONTAINER_STARTED); err != nil {
		return fmt.Errorf("container: %w", err)
	}
	initConn.Close()

	// Accept connection from the host to continue the creation of this container.
	conn, err := listener.Accept()
	if err != nil {
		return fmt.Errorf("container: accept error: %w", err)
	}
	defer conn.Close()

	// TODO: send errors to the host.

	rootfs := container.Rootfs()
	if _, err := os.Stat(rootfs); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("container: rootfs does not exist: %w", err)
	}

	noPivotRoot, _ := cmd.Flags().GetBool("no-pivot")
	mountFlag := syscall.MS_PRIVATE
	if noPivotRoot {
		mountFlag = syscall.MS_SLAVE
	}

	// Prevent mount propagation back to other namespaces.
	if err := syscall.Mount("", "/", "", uintptr(mountFlag|syscall.MS_REC), ""); err != nil {
		return fmt.Errorf("container: failed to prevent mount propagation: %w", err)
	}

	if !noPivotRoot {
		// This seems to be needed for `pivot_root`.
		if err := syscall.Mount(rootfs, rootfs, "bind", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
			return fmt.Errorf("container: failed to bind-mount rootfs: %w", err)
		}
	}

	mounts := container.Spec().Mounts

	logrus.WithFields(logrus.Fields{
		"id":     container.ID(),
		"rootfs": rootfs,
		"mounts": mounts,
	}).Debug("container: mount")

	for _, m := range mounts {
		// Create destination if it does not exist yet.
		dest := filepath.Join(rootfs, m.Destination)
		if _, err := os.Stat(dest); os.IsNotExist(err) {
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return fmt.Errorf("container: failed to create directory: %w", err)
			}
		}

		// TODO: add support for all `m.Options`
		flags := syscall.MS_NOEXEC | syscall.MS_NOSUID | syscall.MS_NODEV

		// HACK: this is mainly used to support default "rootless" specs (created
		// with `runc spec --rootless`).
		if len(m.Options) > 0 && m.Options[0] == "rbind" {
			m.Type = "bind"
			flags |= unix.MS_REC
		}

		if m.Type == "bind" {
			flags |= syscall.MS_BIND
		}

		data := ""
		switch m.Destination {
		case "/dev", "/run":
			flags = syscall.MS_NOSUID | syscall.MS_STRICTATIME
			data = "mode=755,size=65536k"
		case "/dev/pts":
			flags &= ^syscall.MS_NODEV
			data = "newinstance,ptmxmode=0666,mode=0620"
		case "/dev/shm":
			data = "mode=1777,size=65536k"
		case "/sys", "/sys/fs/cgroup":
			flags |= syscall.MS_RDONLY
		}

		if err := syscall.Mount(m.Source, dest, m.Type, uintptr(flags), data); err != nil {
			logrus.WithFields(logrus.Fields{
				"id":          container.ID(),
				"source":      m.Source,
				"destination": dest,
				"type":        m.Type,
				"options":     m.Options,
				"error":       err,
			}).Error("container: failed to mount filesystem")

			// TODO: handle `cgroup`
			if !errors.Is(err, syscall.EPERM) {
				return fmt.Errorf("container: failed to mount: %w", err)
			}
		}
	}

	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config-linux.md#default-devices
	for _, dev := range []string{
		"/dev/null",
		"/dev/zero",
		"/dev/full",
		"/dev/random",
		"/dev/urandom",
		"/dev/tty",
	} {
		dest := filepath.Join(rootfs, dev)

		f, err := os.Create(dest)
		if err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("container: failed to create device destination: %w", err)
		}
		if f != nil {
			f.Close()
		}

		if err := syscall.Mount(dev, dest, "bind", unix.MS_BIND, ""); err != nil {
			return fmt.Errorf("container: failed to mount device: %w", err)
		}
	}

	for _, link := range [][2]string{
		{"/proc/self/fd", "/dev/fd"},
		{"/proc/self/fd/0", "/dev/stdin"},
		{"/proc/self/fd/1", "/dev/stdout"},
		{"/proc/self/fd/2", "/dev/stderr"},
	} {
		src := link[0]
		dst := filepath.Join(rootfs, link[1])

		if err := os.Symlink(src, dst); err != nil && !errors.Is(err, fs.ErrExist) {
			return fmt.Errorf("container: failed to create symlink: %w", err)
		}
	}

	// if container.Spec().Process.Terminal {
	// 	TODO: `/dev/console` is set up if terminal is enabled in the config by bind mounting the pseudoterminal pty to `/dev/console`.
	// }

	// TODO: create symlink for `/dev/ptmx`

	// TODO: linux devices

	// Notify the host that we are about to execute `pivot_root`.
	if err := ipc.SendMessage(conn, ipc.CONTAINER_BEFORE_PIVOT); err != nil {
		return fmt.Errorf("container: %w", err)
	}
	if err := ipc.AwaitMessage(conn, ipc.OK); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	// Hooks to be run after the container has been created but before pivot_root or any equivalent operation has been called. These hooks MUST be called after the `CreateRuntime` hooks.
	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#createcontainer-hooks
	if err := container.ExecuteHooks("CreateContainer"); err != nil {
		logrus.WithError(err).Error("container: CreateContainer hook failed")
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("container: pivot root")

	// Change root filesystem.
	if noPivotRoot {
		if err := syscall.Chroot(rootfs); err != nil {
			return fmt.Errorf("container: failed to change root filesystem: %w", err)
		}
	} else {
		pivotDir := filepath.Join(rootfs, ".pivot_root")
		if err := os.Mkdir(pivotDir, 0o777); err != nil {
			return fmt.Errorf("container: failed to create '.pivot_root': %w", err)
		}
		if err := syscall.PivotRoot(rootfs, pivotDir); err != nil {
			return fmt.Errorf("container: pivot_root failed: %w", err)
		}
		if err := syscall.Chdir("/"); err != nil {
			return fmt.Errorf("container: chdir failed: %w", err)
		}
		pivotDir = filepath.Join("/", ".pivot_root")
		if err := syscall.Unmount(pivotDir, syscall.MNT_DETACH); err != nil {
			return fmt.Errorf("container: failed to unmount '.pivot_root': %w", err)
		}
		os.Remove(pivotDir)
	}

	// Change current working directory.
	if err := syscall.Chdir(container.Spec().Process.Cwd); err != nil {
		return fmt.Errorf("container: failed to change directory: %w", err)
	}

	// Set up new hostname.
	if err := syscall.Sethostname([]byte(container.Spec().Hostname)); err != nil {
		return fmt.Errorf("container: failed to set hostname: %w", err)
	}

	// At this point, the container has been created and when the host receives the message below, it will exits (success).
	if err := ipc.SendMessage(conn, ipc.CONTAINER_WAIT_START); err != nil {
		return fmt.Errorf("container: %w", err)
	}
	conn.Close()

	// Wait until the "start" command connects to this container in order start the container process.
	conn, err = listener.Accept()
	if err != nil {
		return fmt.Errorf("container: accept error: %w", err)
	}
	defer conn.Close()

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("container: waiting for start command")

	if err := ipc.AwaitMessage(conn, ipc.START_CONTAINER); err != nil {
		return fmt.Errorf("container: %w", err)
	}
	conn.Close()
	listener.Close()

	// Hooks to be run after the start operation is called but before the container process is started.
	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#startcontainer-hooks
	if err := container.ExecuteHooks("StartContainer"); err != nil {
		logrus.WithError(err).Error("container: StartContainer hook failed")
	}

	process := container.Spec().Process

	logrus.WithFields(logrus.Fields{
		"id":          container.ID(),
		"processArgs": process.Args,
	}).Info("container: executing process")

	argv0, err := exec.LookPath(process.Args[0])
	if err != nil {
		return fmt.Errorf("container: failed to retrieve executable: %w", err)
	}

	if err := syscall.Exec(argv0, process.Args, process.Env); err != nil {
		return fmt.Errorf("container: failed to exec %v: %w", process.Args, err)
	}

	return nil
}
