package yacs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/cmd"
	"github.com/willdurand/containers/internal/yacs/log"
	"github.com/willdurand/containers/thirdparty/runc/libcontainer/utils"
	"golang.org/x/sys/unix"
)

const (
	consoleSocketName    = "console.sock"
	containerPidFileName = "container.pid"
)

// createContainer creates a new container when the shim is started.
//
// The container is created but not started. This function also creates pipes to
// capture the container `stdout` and `stderr` streams and write their contents
// to files.
func (y *Yacs) createContainer() {
	// Create FIFOs for the container standard IOs.
	for _, name := range []string{"0", "1", "2"} {
		if err := unix.Mkfifo(filepath.Join(y.stdioDir, name), 0o600); err != nil && !errors.Is(err, fs.ErrExist) {
			y.containerReady <- fmt.Errorf("mkfifo: %w", err)
			return
		}
	}

	// We use `O_RDWR` to get non-blocking behavior, see:
	// https://github.com/golang/go/issues/33050#issuecomment-510308419
	sin, err := os.OpenFile(filepath.Join(y.stdioDir, "0"), os.O_RDWR, 0)
	if err != nil {
		y.containerReady <- fmt.Errorf("open stdin: %w", err)
		return
	}
	defer closeFifo(sin)

	sout, err := os.OpenFile(filepath.Join(y.stdioDir, "1"), os.O_RDWR, 0)
	if err != nil {
		y.containerReady <- fmt.Errorf("open stdout: %w", err)
		return
	}
	defer closeFifo(sout)

	serr, err := os.OpenFile(filepath.Join(y.stdioDir, "2"), os.O_RDWR, 0)
	if err != nil {
		y.containerReady <- fmt.Errorf("open stderr: %w", err)
		return
	}
	defer closeFifo(serr)

	// Prepare the arguments for the OCI runtime.
	runtimeArgs := append(
		[]string{y.runtime},
		append(y.runtimeArgs(), []string{
			"create", y.containerID,
			"--bundle", y.bundleDir,
			"--pid-file", y.containerPidFilePath(),
		}...)...,
	)
	if y.containerSpec.Process.Terminal {
		runtimeArgs = append(runtimeArgs, "--console-socket", y.consoleSocketPath())
	}

	// By default, we pass the standard input but the outputs are configured
	// depending on whether the container should create a PTY or not.
	createCommand := exec.Cmd{
		Path: y.runtimePath,
		Args: runtimeArgs,
	}

	// When the container should create a terminal, the shim should open a unix
	// socket and wait until it receives a file descriptor that corresponds to
	// the PTY "master" end.
	if y.containerSpec.Process.Terminal {
		ln, err := net.Listen("unix", y.consoleSocketPath())
		if err != nil {
			y.containerReady <- fmt.Errorf("listen (console socket): %w", err)
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

			// Now we can redirect the streams: first the standard input to the PTY
			// input, then the PTY output to the standard output.
			go io.Copy(ptm, sin)
			go io.Copy(sout, ptm)
		}()
	} else {
		// We only use the log file when the container didn't set up a terminal
		// because that's already complicated enough. That being said, maybe we
		// should log the PTY output as well in the future?
		logFile, err := log.NewFile(y.containerLogFilePath)
		if err != nil {
			y.containerReady <- fmt.Errorf("open (log file): %w", err)
			return
		}
		defer logFile.Close()

		// We create a pipe to pump the stdout from the container and then we write
		// the content to both the log file and the stdout FIFO.
		outRead, outWrite, err := os.Pipe()
		if err != nil {
			y.containerReady <- fmt.Errorf("stdout pipe: %w", err)
			return
		}
		defer outWrite.Close()

		createCommand.Stdout = outWrite
		go copyStd("stdout", outRead, logFile, sout)

		// We create a pipe to pump the stderr from the container and then we write
		// the content to both the log file and the stderr FIFO.
		errRead, errWrite, err := os.Pipe()
		if err != nil {
			y.containerReady <- fmt.Errorf("stderr pipe: %w", err)
			return
		}
		defer errWrite.Close()

		createCommand.Stderr = errWrite
		go copyStd("stderr", errRead, logFile, serr)

		inRead, inWrite, err := os.Pipe()
		if err != nil {
			y.containerReady <- fmt.Errorf("stdin pipe: %w", err)
			return
		}
		defer inRead.Close()

		createCommand.Stdin = inRead
		go func() {
			defer inWrite.Close()

			scanner := bufio.NewScanner(sin)
			for scanner.Scan() {
				data := scanner.Bytes()
				if bytes.Equal(data, []byte("THIS_IS_NOT_HOW_WE_SHOULD_CLOSE_A_PIPE")) {
					break
				}
				inWrite.Write(data)
				inWrite.Write([]byte("\n"))
			}
		}()
	}

	logrus.WithFields(logrus.Fields{
		"command": createCommand.String(),
	}).Info("creating container")

	if err := createCommand.Run(); err != nil {
		y.containerReady <- maybeReturnRuntimeError(y, err)
		return
	}

	logrus.Debug("container created")

	// The runtime should have written the container's PID to a file because
	// that's how the runtime passes this value to the shim. The shim needs the
	// PID to be able to interact with the container directly.
	containerPidFilePath := y.containerPidFilePath()
	data, err := os.ReadFile(containerPidFilePath)
	if err != nil {
		logrus.WithError(err).Panicf("failed to read '%s'", containerPidFilePath)
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logrus.WithError(err).Panicf("failed to parse pid from '%s'", containerPidFilePath)
	}

	// At this point, the shim knows that the runtime has successfully created a
	// container. The shim's API can be used to interact with the container now.
	y.setContainerStatus(&ContainerStatus{PID: containerPid})
	y.containerReady <- nil

	// Wait for the termination of the container process.
	var wstatus syscall.WaitStatus
	var rusage syscall.Rusage
	_, err = syscall.Wait4(containerPid, &wstatus, 0, &rusage)
	if err != nil {
		logrus.WithError(err).Panic("wait4() failed")
	}

	y.setContainerStatus(&ContainerStatus{
		PID:        containerPid,
		WaitStatus: &wstatus,
	})

	logrus.WithFields(logrus.Fields{
		"exitStatus": y.containerStatus.ExitStatus(),
	}).Info("container exited")

	// Close stdio streams in case a container manager is attached (this will
	// notify this manager that the container has exited).
	sin.Close()
	sout.Close()
	serr.Close()

	if y.exitCommand != "" {
		exit := exec.Command(y.exitCommand, y.exitCommandArgs...)
		logrus.WithField("command", exit.String()).Debug("execute exit command")

		if err := cmd.Run(exit); err != nil {
			logrus.WithError(err).Warn("exit command failed")
		}
	}
}

// containerPidFilePath returns the path to the file that contains the PID of
// the container. Usually, this path should be passed to the OCI runtime with a
// CLI flag (`--pid-file`).
func (y *Yacs) containerPidFilePath() string {
	return filepath.Join(y.baseDir, containerPidFileName)
}

// consoleSocketPath returns the path to the console socket that is used by the
// container when it must create a PTY.
func (y *Yacs) consoleSocketPath() string {
	return filepath.Join(y.baseDir, consoleSocketName)
}

// setContainerStatus sets an instance of `ContainerStatus` to the shim
// configuration.
func (y *Yacs) setContainerStatus(status *ContainerStatus) {
	y.containerStatus = status
}

// copyStd copies the content of `src` into the provided log file and FIFO. This
// is a ugly version of a `MultiWriter` + `Copy()` for Yacs.
func copyStd(name string, src *os.File, logFile *log.LogFile, fifo *os.File) {
	defer src.Close()

	scanner := bufio.NewScanner(src)
	for scanner.Scan() {
		m := scanner.Text()
		fifo.WriteString(m + "\n")
		logFile.WriteMessage(name, m)
	}
}

// maybeReturnRuntimeError reads the runtime log file in order to return the
// last message logged by the runtime as an error. If anything goes wrong, the
// original error is returned instead.
func maybeReturnRuntimeError(y *Yacs, originalError error) error {
	logFile, err := os.Open(y.runtimeLogFilePath())
	if err != nil {
		return originalError
	}
	defer logFile.Close()

	// Let's consider the two last lines in the log file.
	lastLines := [][]byte{[]byte(""), []byte("")}

	scanner := bufio.NewScanner(logFile)
	for scanner.Scan() {
		lastLines[0] = lastLines[1]
		lastLines[1] = scanner.Bytes()
	}

	for _, lastLine := range lastLines {
		log := make(map[string]string)
		if err := json.Unmarshal(lastLine, &log); err != nil {
			return originalError
		}

		if log["level"] != "error" {
			continue
		}

		if msg := log["msg"]; msg != "" {
			return errors.New(msg)
		}
	}

	return originalError
}

func closeFifo(f *os.File) {
	f.Close()

	if err := os.Remove(f.Name()); err != nil {
		logrus.WithError(err).WithField("name", f.Name()).Warn("failed to remove FIFO")
	}
}
