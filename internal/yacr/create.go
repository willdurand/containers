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

	// TODO: make sure that runtimespec.Version is supported

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
	for _, ns := range container.Spec().Linux.Namespaces {
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

	var uidMappings []syscall.SysProcIDMap
	for _, m := range container.Spec().Linux.UIDMappings {
		uidMappings = append(uidMappings, syscall.SysProcIDMap{
			ContainerID: int(m.ContainerID),
			HostID:      int(m.HostID),
			Size:        int(m.Size),
		})
	}

	var gidMappings []syscall.SysProcIDMap
	for _, m := range container.Spec().Linux.GIDMappings {
		gidMappings = append(gidMappings, syscall.SysProcIDMap{
			ContainerID: int(m.ContainerID),
			HostID:      int(m.HostID),
			Size:        int(m.Size),
		})
	}

	self, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to retrieve executable: %w", err)
	}
	containerProcess := &exec.Cmd{
		Path: self,
		Args: append([]string{"yacr"}, containerArgs...),
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags:  uintptr(cloneFlags),
			UidMappings: uidMappings,
			GidMappings: gidMappings,
		},
	}

	logrus.WithFields(logrus.Fields{
		"id":      container.ID(),
		"process": containerProcess.String(),
	}).Debug("container process configured")

	if container.Spec().Process.Terminal {
		// See: https://github.com/opencontainers/runc/blob/016a0d29d1750180b2a619fc70d6fe0d80111be0/docs/terminals.md#detached-new-terminal
		if err := ipc.EnsureValidSockAddr(opts.ConsoleSocket, true); err != nil {
			return err
		}

		logrus.WithFields(logrus.Fields{
			"id":            container.ID(),
			"consoleSocket": opts.ConsoleSocket,
		}).Debug("start container process with pty")

		ptmx, err := pty.Start(containerProcess)
		if err != nil {
			return fmt.Errorf("failed to create container (1): %w", err)
		}
		defer ptmx.Close()

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
		oob := unix.UnixRights(int(ptmx.Fd()))
		uc.WriteMsgUnix([]byte(ptmx.Name()), oob, nil)
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

	// Wait until the container has started.
	initConn, err := initListener.Accept()
	if err != nil {
		return fmt.Errorf("init accept error: %w", err)
	}
	defer initConn.Close()

	if err := ipc.AwaitMessage(initConn, ipc.CONTAINER_STARTED); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("container successfully started")

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

	containerPid := containerProcess.Process.Pid

	// Write the container PID to the pid file if supplied.
	if opts.PidFile != "" {
		if err := ioutil.WriteFile(opts.PidFile, []byte(strconv.FormatInt(int64(containerPid), 10)), 0o644); err != nil {
			return fmt.Errorf("failed to write to pid file: %w", err)
		}
	}

	// Update state.
	if err := container.SaveAsCreated(containerPid); err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Info("ok")

	return nil
}