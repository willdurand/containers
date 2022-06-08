package cmd

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
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/cmd/yacr/containers"
	"github.com/willdurand/containers/cmd/yacr/ipc"
	"golang.org/x/sys/unix"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/spf13/cobra"
)

func init() {
	cmd := &cobra.Command{
		Use:          "create <id>",
		Short:        "Create a container",
		SilenceUsage: true,
		RunE:         create,
		Args:         cobra.ExactArgs(1),
	}
	cmd.PersistentFlags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	cmd.MarkFlagRequired("bundle")
	cmd.Flags().String("pid-file", "", "specify the file to write the process id to")
	cmd.Flags().String("console-socket", "", "console unix socket used to pass a PTY descriptor")
	cmd.Flags().Bool("no-pivot", false, "do not use pivot root to jail process inside rootfs")
	rootCmd.AddCommand(cmd)

	containerCmd := &cobra.Command{
		Use:          "container <id>",
		SilenceUsage: true,
		RunE:         container,
		Hidden:       true,
		Args:         cobra.ExactArgs(1),
	}
	cmd.AddCommand(containerCmd)
}

func create(cmd *cobra.Command, args []string) error {
	bundle, err := cmd.Flags().GetString("bundle")
	if err != nil || bundle == "" {
		return fmt.Errorf("create: %w", err)
	}

	rootDir, _ := cmd.Flags().GetString("root")
	container, err := containers.New(rootDir, args[0], bundle)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	// TODO: make sure that specs.Version is supported

	// TODO: error when there is no linux configuration

	if err := container.Save(); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("create: new container created")

	// Create an initial socket that we pass to the container. When the container starts, it should inform the host (this process). After that, we discard this socket and connect to the container's socket, which is needed for the `start` command (at least).
	initSockAddr, err := container.GetInitSockAddr(false)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	initListener, err := net.Listen("unix", initSockAddr)
	if err != nil {
		return fmt.Errorf("create: listen error: %w", err)
	}
	defer initListener.Close()

	// Prepare a command to re-execute itself in order to create the container process.
	logFile, _ := cmd.Flags().GetString("log")
	if logFile != "" {
		args = append([]string{"--log", logFile}, args...)
	}

	logFormat, _ := cmd.Flags().GetString("log-format")
	if logFormat != "" {
		args = append([]string{"--log-format", logFormat}, args...)
	}

	if debug, _ := cmd.Flags().GetBool("debug"); debug {
		args = append([]string{"--debug"}, args...)
	}

	if noPivot, _ := cmd.Flags().GetBool("no-pivot"); noPivot {
		args = append([]string{"--no-pivot"}, args...)
	}

	containerArgs := append(
		[]string{"create", "container", "--root", rootDir},
		args...,
	)

	var cloneFlags uintptr
	for _, ns := range container.Spec().Linux.Namespaces {
		switch ns.Type {
		case specs.UTSNamespace:
			cloneFlags |= syscall.CLONE_NEWUTS
		case specs.PIDNamespace:
			cloneFlags |= syscall.CLONE_NEWPID
		case specs.MountNamespace:
			cloneFlags |= syscall.CLONE_NEWNS
		case specs.UserNamespace:
			cloneFlags |= syscall.CLONE_NEWUSER
		case specs.NetworkNamespace:
			cloneFlags |= syscall.CLONE_NEWNET
		case specs.IPCNamespace:
			cloneFlags |= syscall.CLONE_NEWIPC
		default:
			return fmt.Errorf("create: unsupported namespace: %s", ns.Type)
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
		return fmt.Errorf("create: failed to retrieve executable: %w", err)
	}
	containerProcess := &exec.Cmd{
		Path: self,
		Args: append([]string{programName}, containerArgs...),
		SysProcAttr: &syscall.SysProcAttr{
			Cloneflags:  uintptr(cloneFlags),
			UidMappings: uidMappings,
			GidMappings: gidMappings,
		},
	}

	logrus.WithFields(logrus.Fields{
		"id":      container.ID(),
		"process": containerProcess.String(),
	}).Debug("create: container process configured")

	if container.Spec().Process.Terminal {
		// See: https://github.com/opencontainers/runc/blob/016a0d29d1750180b2a619fc70d6fe0d80111be0/docs/terminals.md#detached-new-terminal
		consoleSocket, _ := cmd.Flags().GetString("console-socket")
		if err := ipc.EnsureValidSockAddr(consoleSocket, true); err != nil {
			return fmt.Errorf("create: %w", err)
		}

		logrus.WithFields(logrus.Fields{
			"id":            container.ID(),
			"consoleSocket": consoleSocket,
		}).Debug("create: start container process with pty")

		ptmx, err := pty.Start(containerProcess)
		if err != nil {
			return fmt.Errorf("create: failed to create container (1): %w", err)
		}
		defer ptmx.Close()

		// Connect to the socket in order to send the PTY file descriptor.
		conn, err := net.Dial("unix", consoleSocket)
		if err != nil {
			return fmt.Errorf("create: failed to dial console socket: %w", err)
		}
		defer conn.Close()

		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return errors.New("create: failed to cast unix socket")
		}
		defer uc.Close()

		// Send file descriptor over socket.
		oob := unix.UnixRights(int(ptmx.Fd()))
		uc.WriteMsgUnix([]byte(ptmx.Name()), oob, nil)
	} else {
		logrus.WithFields(logrus.Fields{
			"id": container.ID(),
		}).Debug("create: start container process without pty")

		containerProcess.Stdin = os.Stdin
		containerProcess.Stdout = os.Stdout
		containerProcess.Stderr = os.Stderr

		if err := containerProcess.Start(); err != nil {
			return fmt.Errorf("create: failed to create container (2): %w", err)
		}
	}

	// Wait until the container has started.
	initConn, err := initListener.Accept()
	if err != nil {
		return fmt.Errorf("create: init accept error: %w", err)
	}
	defer initConn.Close()

	if err := ipc.AwaitMessage(initConn, ipc.CONTAINER_STARTED); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Debug("create: container successfully started")

	initConn.Close()
	initListener.Close()
	syscall.Unlink(initSockAddr)

	// Connect to the container.
	sockAddr, err := container.GetSockAddr(true)
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}
	conn, err := net.Dial("unix", sockAddr)
	if err != nil {
		return fmt.Errorf("create: failed to dial container socket: %w", err)
	}
	defer conn.Close()

	// Wait until the container reached the "before pivot_root" step so that we can run `CreateRuntime` hooks.
	if err := ipc.AwaitMessage(conn, ipc.CONTAINER_BEFORE_PIVOT); err != nil {
		return fmt.Errorf("create: before_pivot: %w", err)
	}

	// Hooks to be run after the container has been created but before `pivot_root`.
	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#createruntime-hooks
	if err := container.ExecuteHooks("CreateRuntime"); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	// Notify the container that it can continue its initialization.
	if err := ipc.SendMessage(conn, ipc.OK); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	// Wait until the container is ready (i.e. the container waits for the "start" command).
	if err := ipc.AwaitMessage(conn, ipc.CONTAINER_WAIT_START); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	containerPid := containerProcess.Process.Pid

	// Write the container PID to the pid file if supplied.
	if pidFile, _ := cmd.Flags().GetString("pid-file"); pidFile != "" {
		if err := ioutil.WriteFile(pidFile, []byte(strconv.FormatInt(int64(containerPid), 10)), 0o644); err != nil {
			return fmt.Errorf("create: failed to write to pid file: %w", err)
		}
	}

	// Update state.
	if err := container.SaveAsCreated(containerPid); err != nil {
		return fmt.Errorf("create: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Info("create: ok")

	return nil
}
