package yacs

import (
	"bytes"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yacs/log"
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

		if y.exitCommand != "" {
			exit := exec.Command(y.exitCommand, y.exitCommandArgs...)
			logger.WithField("command", exit.String()).Debug("execute exit command")

			if err := exit.Run(); err != nil {
				logger.WithError(err).Warn("exit command failed")
			}
		}
	}()

	logFile, err := log.NewFile(y.ContainerLogFile)
	if err != nil {
		logger.WithError(err).Panic("failed to create log file")
	}
	defer logFile.Close()

	outRead, outWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Panic("failed to create out pipe")
	}
	defer outRead.Close()
	defer outWrite.Close()

	go logFile.WriteStream(outRead, "stdout")

	errRead, errWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Panic("failed to create err pipe")
	}
	defer errRead.Close()
	defer errWrite.Close()

	go logFile.WriteStream(errRead, "stderr")

	runtimeArgs := append(
		[]string{y.runtime},
		append(y.runtimeArgs(), []string{
			"create", y.ContainerID,
			"--bundle", y.bundle,
			"--pid-file", y.containerPidFile(),
		}...)...,
	)

	runtimeCommand := &exec.Cmd{
		Path:   y.runtimePath,
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
	data, err := os.ReadFile(y.containerPidFile())
	if err != nil {
		logger.WithError(err).Panicf("failed to read '%s'", y.containerPidFile())
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logger.WithError(err).Panicf("failed to parse pid from '%s'", y.containerPidFile())
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
}
