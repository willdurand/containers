package main

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacs/config"
)

// createContainer creates a new container when the shim is started.
//
// The container is created but not started. This function also creates pipes to
// capture the container `stdout` and `stderr` streams and write their contents
// to files.
func createContainer(cfg *config.ShimConfig, logger *logrus.Entry, cmd *cobra.Command) {
	outRead, outWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Fatal("failed to create out pipe")
	}
	defer outRead.Close()
	defer outWrite.Close()

	// Store the container's stdout to a file.
	outFile, _ := os.OpenFile(cfg.StdoutFileName(), os.O_CREATE|os.O_WRONLY, 0o644)
	go io.Copy(outFile, outRead)

	errRead, errWrite, err := os.Pipe()
	if err != nil {
		logger.WithError(err).Fatal("failed to create err pipe")
	}
	defer errRead.Close()
	defer errWrite.Close()

	// Store the container's stderr to a file.
	errFile, _ := os.OpenFile(cfg.StderrFileName(), os.O_CREATE|os.O_WRONLY, 0o644)
	go io.Copy(errFile, errRead)

	runtimeArgs := append(
		[]string{cfg.Runtime()},
		append(cfg.RuntimeArgs(), []string{
			"create", cfg.ContainerID(),
			"--bundle", cfg.Bundle(),
			"--pid-file", cfg.ContainerPidFileName(),
		}...)...,
	)

	process := &exec.Cmd{
		Path:   cfg.RuntimePath(),
		Args:   runtimeArgs,
		Stdin:  nil,
		Stdout: outWrite,
		Stderr: errWrite,
	}

	logger.WithFields(logrus.Fields{
		"command": process.String(),
	}).Info("creating container")

	if err := process.Run(); err != nil {
		logger.WithError(err).Fatal("failed to create container")
	}
	logger.Debug("container created")

	// The runtime should have written the container's PID to a file because
	// that's how the runtime passes this value to the shim. The shim needs the
	// PID to be able to interact with the container directly.
	data, err := os.ReadFile(cfg.ContainerPidFileName())
	if err != nil {
		logger.WithError(err).Fatalf("failed to read '%s'", cfg.ContainerPidFileName())
	}
	containerPid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		logger.WithError(err).Fatalf("failed to parse pid from '%s'", cfg.ContainerPidFileName())
	}

	// At this point, the shim knows that the runtime has successfully created a
	// container. The shim's API can be used to interact with the container now.
	cfg.SetContainerStatus(&config.ContainerStatus{PID: containerPid})

	// Wait for the termination of the container process.
	var wstatus syscall.WaitStatus
	var rusage syscall.Rusage
	_, err = syscall.Wait4(containerPid, &wstatus, 0, &rusage)
	if err != nil {
		logger.WithError(err).Fatal("wait4() failed")
	}

	cfg.SetContainerStatus(&config.ContainerStatus{
		PID:        containerPid,
		WaitStatus: &wstatus,
	})

	logger.WithFields(logrus.Fields{
		"exitStatus": cfg.ContainerStatus().ExitStatus(),
	}).Info("container exited")
}
