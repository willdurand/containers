package yacs

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
)

// CreateContainer creates a new container when the shim is started.
//
// The container is created but not started. This function also creates pipes to
// capture the container `stdout` and `stderr` streams and write their contents
// to files.
func (y *Yacs) CreateContainer(logger *logrus.Entry) {
	defer func() {
		if y.containerStatus == nil {
			args := append(y.runtimeArgs(), []string{
				"delete", y.ContainerID, "--force",
			}...)
			if err := exec.Command(y.runtimePath, args...).Run(); err != nil {
				logger.WithError(err).Error("failed to force delete container")
			}

			y.Destroy()
		}
	}()

	outRead, outWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Panic("failed to create out pipe")
	}
	defer outRead.Close()
	defer outWrite.Close()

	// We'll store the container's stdout/stderr to a file (using a JSON object
	// per line). For now we use a single pipe for both streams, which is simpler
	// but prevents us from knowing the source of each message.
	logFile, err := os.OpenFile(y.ContainerLogFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		logger.WithError(err).Panic("failed to create log file")
	}
	go func() {
		scanner := bufio.NewScanner(outRead)
		for scanner.Scan() {
			data, err := json.Marshal(map[string]interface{}{
				"t": time.Now(),
				"m": scanner.Text(),
				"s": "",
			})
			if err == nil {
				if _, err := logFile.Write(append(data, '\n')); err != nil {
					logrus.WithError(err).Warn("failed to write to container log file")
				}
			}
		}
	}()

	runtimeArgs := append(
		[]string{y.runtime},
		append(y.runtimeArgs(), []string{
			"create", y.ContainerID,
			"--bundle", y.bundle,
			"--pid-file", y.containerPidFileName(),
		}...)...,
	)

	runtimeCommand := &exec.Cmd{
		Path:   y.runtimePath,
		Args:   runtimeArgs,
		Stdin:  nil,
		Stdout: outWrite,
		Stderr: outWrite,
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
	data, err := os.ReadFile(y.containerPidFileName())
	if err != nil {
		logger.WithError(err).Panicf("failed to read '%s'", y.containerPidFileName())
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logger.WithError(err).Panicf("failed to parse pid from '%s'", y.containerPidFileName())
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
		exit := exec.Cmd{
			Path:   y.exitCommand,
			Args:   append([]string{y.exitCommand}, y.exitCommandArgs...),
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
