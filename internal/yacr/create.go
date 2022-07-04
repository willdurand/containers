package yacr

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/creack/pty"
	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/cmd"
	"github.com/willdurand/containers/internal/yacr/container"
	"github.com/willdurand/containers/internal/yacr/ipc"
	"golang.org/x/sys/unix"
)

type CreateOpts struct {
	ID            string
	Bundle        string
	PidFile       string
	ConsoleSocket string
	NoPivot       bool
	LogFile       string
	LogFormat     string
	Debug         bool
}

func Create(rootDir string, opts CreateOpts) error {
	if opts.Bundle == "" {
		return fmt.Errorf("invalid bundle")
	}

	container, err := container.New(rootDir, opts.ID, opts.Bundle)
	if err != nil {
		return err
	}

	// TODO: error when there is no linux configuration

	if err := container.Save(); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("new container created")

	// Create an initial socket that we pass to the container. When the
	// container starts, it should inform the host (this process). After that,
	// we discard this socket and connect to the container's socket, which is
	// needed for the `start` command (at least).
	initSockAddr, err := container.GetInitSockAddr(false)
	if err != nil {
		return err
	}
	initListener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("listen error: %w", err)
	}
	defer initListener.Close()

	// Prepare a command to re-execute itself in order to create the container
	// process.
	containerArgs := []string{
		"create", "container", opts.ID,
		"--root", rootDir,
		"--bundle", opts.Bundle,
	}
	if opts.LogFile != "" {
		containerArgs = append([]string{"--log", opts.LogFile}, containerArgs...)
	}
	if opts.LogFormat != "" {
		containerArgs = append([]string{"--log-format", opts.LogFormat}, containerArgs...)
	}
	if opts.Debug {
		containerArgs = append([]string{"--debug"}, containerArgs...)
	}
	if opts.NoPivot {
		containerArgs = append([]string{"--no-pivot"}, containerArgs...)
	}

	var cloneFlags uintptr
	for _, ns := range container.Spec.Linux.Namespaces {
		switch ns.Type {
		case runtimespec.UTSNamespace:
			cloneFlags |= syscall.CLONE_NEWUTS
		case runtimespec.PIDNamespace:
			cloneFlags |= syscall.CLONE_NEWPID
		case runtimespec.MountNamespace:
			cloneFlags |= syscall.CLONE_NEWNS
		case runtimespec.UserNamespace:
			cloneFlags |= syscall.CLONE_NEWUSER
		case runtimespec.NetworkNamespace:
			cloneFlags |= syscall.CLONE_NEWNET
		case runtimespec.IPCNamespace:
			cloneFlags |= syscall.CLONE_NEWIPC
		default:
			return fmt.Errorf("unsupported namespace: %s", ns.Type)
		}
	}

	env := os.Environ()
	if cloneFlags&syscall.CLONE_NEWUSER != syscall.CLONE_NEWUSER {
		// When we don't have a user namespace, there is no need to re-exec because
		// we won't configure the uid/gid maps.
		env = append(env, "_YACR_CONTAINER_REEXEC=1")
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to retrieve executable: %w", err)
	}
	containerProcess := &exec.Cmd{
		Path: self,
		Args: append([]string{"yacr"}, containerArgs...),
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags: uintptr(cloneFlags),
		},
		Env: env,
	}

	logrus.WithFields(logrus.Fields{
		"id":      container.ID(),
		"process": containerProcess.String(),
	}).Debug("container process configured")

	if container.Spec.Process.Terminal {
		// See: https://github.com/opencontainers/runc/blob/016a0d29d1750180b2a619fc70d6fe0d80111be0/docs/terminals.md#detached-new-terminal
		if err := ipc.EnsureValidSockAddr(opts.ConsoleSocket, true); err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"id":            container.ID(),
			"consoleSocket": opts.ConsoleSocket,
		}).Debug("start container process with pty")

		ptm, err := pty.Start(containerProcess)
		if err != nil {
			return fmt.Errorf("failed to create container (1): %w", err)
		}
		defer ptm.Close()

		// Connect to the socket in order to send the PTY file descriptor.
		conn, err := net.Dial("unix", opts.ConsoleSocket)
		if err != nil {
			return fmt.Errorf("failed to dial console socket: %w", err)
		}
		defer conn.Close()

		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return errors.New("failed to cast unix socket")
		}
		defer uc.Close()

		// Send file descriptor over socket.
		oob := unix.UnixRights(int(ptm.Fd()))
		uc.WriteMsgUnix([]byte(ptm.Name()), oob, nil)
	} else {
		logrus.WithFields(logrus.Fields{
			"id": container.ID(),
		}).Debug("start container process without pty")

		containerProcess.Stdin = os.Stdin
		containerProcess.Stdout = os.Stdout
		containerProcess.Stderr = os.Stderr

		if err := containerProcess.Start(); err != nil {
			return fmt.Errorf("failed to create container (2): %w", err)
		}
	}

	if cloneFlags&syscall.CLONE_NEWUSER == syscall.CLONE_NEWUSER {
		newuidmap, err := exec.LookPath("newuidmap")
		if err != nil {
			return err
		}

		var uidMap []string
		for _, m := range container.Spec.Linux.UIDMappings {
			uidMap = append(uidMap, []string{
				strconv.Itoa(int(m.ContainerID)),
				strconv.Itoa(int(m.HostID)),
				strconv.Itoa(int(m.Size)),
			}...)
		}

		newuidmapCmd := exec.Command(newuidmap, append(
			[]string{strconv.Itoa(containerProcess.Process.Pid)}, uidMap...,
		)...)
		logrus.WithField("command", newuidmapCmd.String()).Debug("configuring uidmap")

		if err := cmd.Run(newuidmapCmd); err != nil {
			return fmt.Errorf("newuidmap failed: %w", err)
		}

		newgidmap, err := exec.LookPath("newgidmap")
		if err != nil {
			return err
		}

		var gidMap []string
		for _, m := range container.Spec.Linux.GIDMappings {
			gidMap = append(gidMap, []string{
				strconv.Itoa(int(m.ContainerID)),
				strconv.Itoa(int(m.HostID)),
				strconv.Itoa(int(m.Size)),
			}...)
		}

		newgidmapCmd := exec.Command(newgidmap, append(
			[]string{strconv.Itoa(containerProcess.Process.Pid)}, gidMap...,
		)...)
		logrus.WithField("command", newgidmapCmd.String()).Debug("configuring gidmap")

		if err := cmd.Run(newgidmapCmd); err != nil {
			return fmt.Errorf("newgidmap failed: %w", err)
		}
	}

	// Wait until the container has "booted".
	initConn, err := initListener.Accept()
	if err != nil {
		return fmt.Errorf("init accept error: %w", err)
	}
	defer initConn.Close()

	if err := ipc.AwaitMessage(initConn, ipc.CONTAINER_BOOTED); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("container booted")

	initConn.Close()
	initListener.Close()
	syscall.Unlink(initSockAddr)

	// Connect to the container.
	sockAddr, err := container.GetSockAddr(true)
	if err != nil {
		return err
	}
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("failed to dial container socket: %w", err)
	}
	defer conn.Close()

	// Wait until the container reached the "before pivot_root" step so that we
	// can run `CreateRuntime` hooks.
	if err := ipc.AwaitMessage(conn, ipc.CONTAINER_BEFORE_PIVOT); err != nil {
		return fmt.Errorf("before_pivot: %w", err)
	}

	containerPid := containerProcess.Process.Pid
	container.SetPid(containerPid)

	// Write the container PID to the pid file if supplied.
	if opts.PidFile != "" {
		if err := ioutil.WriteFile(opts.PidFile, []byte(strconv.Itoa(containerPid)), 0o644); err != nil {
			return fmt.Errorf("failed to write to pid file: %w", err)
		}
	}

	// Hooks to be run after the container has been created but before
	// `pivot_root`.
	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#createruntime-hooks
	if err := container.ExecuteHooks("CreateRuntime"); err != nil {
		return err
	}

	// Notify the container that it can continue its initialization.
	if err := ipc.SendMessage(conn, ipc.OK); err != nil {
		return err
	}

	// Wait until the container is ready (i.e. the container waits for the
	// "start" command).
	if err := ipc.AwaitMessage(conn, ipc.CONTAINER_WAIT_START); err != nil {
		return err
	}

	// Update state.
	if err := container.SaveAsCreated(); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Info("container created")

	return nil
}
