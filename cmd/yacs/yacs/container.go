package yacs

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
)

// CreateContainer creates a new container when the shim is started.
//
// The container is created but not started. This function also creates pipes to
// capture the container `stdout` and `stderr` streams and write their contents
// to files.
func (s *Shim) CreateContainer(logger *logrus.Entry) {
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Panic("failed to create out pipe")
	}
	defer outRead.Close()
	defer outWrite.Close()

	// Store the container's stdout to a file.
	outFile, _ := os.OpenFile(s.stdoutFileName(), os.O_CREATE|os.O_WRONLY, 0o644)
	go io.Copy(outFile, outRead)

	errRead, errWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Panic("failed to create err pipe")
	}
	defer errRead.Close()
	defer errWrite.Close()

	// Store the container's stderr to a file.
	errFile, _ := os.OpenFile(s.stderrFileName(), os.O_CREATE|os.O_WRONLY, 0o644)
	go io.Copy(errFile, errRead)

	runtimeArgs := append(
		[]string{s.runtime},
		append(s.runtimeArgs(), []string{
			"create", s.ContainerID(),
			"--bundle", s.bundle,
			"--pid-file", s.containerPidFileName(),
		}...)...,
	)

	runtimeCommand := &exec.Cmd{
		Path:   s.runtimePath,
		Args:   runtimeArgs,
		Stdin:  nil,
		Stdout: outWrite,
		Stderr: errWrite,
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
	data, err := os.ReadFile(s.containerPidFileName())
	if err != nil {
		logger.WithError(err).Panicf("failed to read '%s'", s.containerPidFileName())
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logger.WithError(err).Panicf("failed to parse pid from '%s'", s.containerPidFileName())
	}

	// At this point, the shim knows that the runtime has successfully created a
	// container. The shim's API can be used to interact with the container now.
	s.setContainerStatus(&ContainerStatus{PID: containerPid})

	// Wait for the termination of the container process.
	var wstatus syscall.WaitStatus
	var rusage syscall.Rusage
	_, err = syscall.Wait4(containerPid, &wstatus, 0, &rusage)
	if err != nil {
		logger.WithError(err).Panic("wait4() failed")
	}

	s.setContainerStatus(&ContainerStatus{
		PID:        containerPid,
		WaitStatus: &wstatus,
	})

	logger.WithFields(logrus.Fields{
		"exitStatus": s.containerStatus.ExitStatus(),
	}).Info("container exited")

	if s.exitCommand != "" {
		exit := exec.Cmd{
			Path:   s.exitCommand,
			Args:   append([]string{s.exitCommand}, s.exitCommandArgs...),
			Stdin:  nil,
			Stdout: nil,
			Stderr: nil,
		}
		logger.WithField("command", exit.String()).Debug("execute exit command")

		if err := exit.Run(); err != nil {
			logger.WithError(err).Warn("exit command failed")
		}
	}
}
