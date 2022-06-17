package yacs

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yacs/log"
	"github.com/willdurand/containers/thirdparty/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
)

// CreateContainer creates a new container when the shim is started.
//
// The container is created but not started. This function also creates pipes to
// capture the container `stdout` and `stderr` streams and write their contents
// to files.
func (y *Yacs) CreateContainer(logger *logrus.Entry) {
	defer func() {
		// In case of an error during the creation of the container, we call the
		// OCI runtime to (force) delete the container.
		if y.containerStatus == nil {
			del := exec.Command(y.runtimePath, append(y.runtimeArgs(), []string{
				"delete", y.ContainerID, "--force",
			}...)...)
			logger.WithField("command", del.String()).Debug("execute delete command")

			if err := del.Run(); err != nil {
				logger.WithError(err).Error("failed to force delete container")
			}

			y.Destroy()
		}
	}()

	// Create FIFOs for the container standard IOs.
	for _, name := range []string{"0", "1", "2"} {
		if err := unix.Mkfifo(filepath.Join(y.stdioDir, name), 0o600); err != nil {
			logger.WithError(err).Panicf("failed to make fifo '%s'", name)
		}
	}

	sin, err := os.OpenFile(filepath.Join(y.stdioDir, "0"), os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		logger.WithError(err).Panic("failed to open stdin fifo")
	}
	defer sin.Close()

	sout, err := os.OpenFile(filepath.Join(y.stdioDir, "1"), os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		logger.WithError(err).Panic("failed to open stdout fifo")
	}
	defer sout.Close()

	serr, err := os.OpenFile(filepath.Join(y.stdioDir, "2"), os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		logger.WithError(err).Panic("failed to open stderr fifo")
	}
	defer serr.Close()

	// Prepare the arguments for the OCI runtime.
	runtimeArgs := append(
		[]string{y.runtime},
		append(y.runtimeArgs(), []string{
			"create", y.ContainerID,
			"--bundle", y.bundleDir,
			"--pid-file", y.containerPidFilePath(),
		}...)...,
	)
	if y.containerSpec.Process.Terminal {
		runtimeArgs = append(runtimeArgs, "--console-socket", y.consoleSocketPath())
	}

	// By default, we pass the standard input but the outputs are configured
	// depending on whether the container should create a PTY or not.
	runtimeCommand := exec.Cmd{
		Path:  y.runtimePath,
		Args:  runtimeArgs,
		Stdin: sin,
	}

	// When the container should create a terminal, the shim should open a unix
	// socket and wait until it receives a file descriptor that corresponds to
	// the PTY "master" end.
	if y.containerSpec.Process.Terminal {
		ln, err := net.Listen("unix", y.consoleSocketPath())
		if err != nil {
			logrus.WithError(err).Panic("failed to listen to console socket")
		}
		defer ln.Close()

		go func() {
			conn, err := ln.Accept()
			if err != nil {
				logrus.WithError(err).Panic("failed to accept connections on console socket")
			}
			defer conn.Close()

			unixconn, ok := conn.(*net.UnixConn)
			if !ok {
				logrus.WithError(err).Panic("failed to cast to unixconn")
			}

			socket, err := unixconn.File()
			if err != nil {
				logrus.WithError(err).Panic("failed to retrieve socket file")
			}
			defer socket.Close()

			ptm, err := utils.RecvFd(socket)
			if err != nil {
				logrus.WithError(err).Panic("failed to receive file descriptor")
			}

			logrus.Debug("got a ptm")

			// Now we can redirect the streams: first the standard input to the PTY
			// input, then the PTY output to the standard output.
			go io.Copy(ptm, sin)
			go io.Copy(sout, ptm)
		}()
	} else {
		// We only use the log file when the container didn't set up a terminal
		// because that's already complicated enough. That being said, maybe we
		// should log the PTY output as well in the future?
		logFile, err := log.NewFile(y.ContainerLogFilePath)
		if err != nil {
			logger.WithError(err).Panic("failed to create log file")
		}
		defer logFile.Close()

		// We create a pipe to pump the stdout from the container and then we write
		// the content to both the log file and the stdout FIFO.
		outRead, outWrite, err := os.Pipe()
		if err != nil {
			logger.WithError(err).Panic("failed to create stdout pipe")
		}
		defer outWrite.Close()

		runtimeCommand.Stdout = outWrite
		go copyStd("stdout", outRead, logFile, sout)

		// We create a pipe to pump the stderr from the container and then we write
		// the content to both the log file and the stderr FIFO.
		errRead, errWrite, err := os.Pipe()
		if err != nil {
			logger.WithError(err).Panic("failed to create stderr pipe")
		}
		defer errWrite.Close()

		runtimeCommand.Stderr = errWrite
		go copyStd("stderr", errRead, logFile, serr)
	}

	logger.WithFields(logrus.Fields{
		"command": runtimeCommand.String(),
	}).Info("creating container")

	if err := runtimeCommand.Run(); err != nil {
		logger.WithError(err).Panic("failed to create container")
	}
	logger.Debug("container created")

	// The runtime should have written the container's PID to a file because
	// that's how the runtime passes this value to the shim. The shim needs the
	// PID to be able to interact with the container directly.
	containerPidFilePath := y.containerPidFilePath()
	data, err := os.ReadFile(containerPidFilePath)
	if err != nil {
		logger.WithError(err).Panicf("failed to read '%s'", containerPidFilePath)
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logger.WithError(err).Panicf("failed to parse pid from '%s'", containerPidFilePath)
	}

	// At this point, the shim knows that the runtime has successfully created a
	// container. The shim's API can be used to interact with the container now.
	y.setContainerStatus(&ContainerStatus{PID: containerPid})

	// Wait for the termination of the container process.
	var wstatus syscall.WaitStatus
	var rusage syscall.Rusage
	_, err = syscall.Wait4(containerPid, &wstatus, 0, &rusage)
	if err != nil {
		logger.WithError(err).Panic("wait4() failed")
	}

	y.setContainerStatus(&ContainerStatus{
		PID:        containerPid,
		WaitStatus: &wstatus,
	})

	logger.WithFields(logrus.Fields{
		"exitStatus": y.containerStatus.ExitStatus(),
	}).Info("container exited")

	if y.exitCommand != "" {
		exit := exec.Command(y.exitCommand, y.exitCommandArgs...)
		logger.WithField("command", exit.String()).Debug("execute exit command")

		if err := exit.Run(); err != nil {
			logger.WithError(err).Warn("exit command failed")
		}
	}
}

func copyStd(name string, src *os.File, logFile *log.LogFile, fifo *os.File) {
	defer src.Close()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		m := scanner.Text()
		fifo.WriteString(m + "\n")
		logFile.WriteMessage(name, m)
	}
}
