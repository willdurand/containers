package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sevlyar/go-daemon"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/cmd"
	"github.com/willdurand/containers/yacs/config"
	"golang.org/x/sys/unix"
)

const (
	programName string = "yacs"
)

// shimCmd represents the shim command (which is the base command).
var shimCmd = cmd.NewRootCommand(
	programName,
	"Yet another container shim",
)

var exitShim = make(chan struct{})

func init() {
	// We want to execute a function by default.
	shimCmd.RunE = shim
	shimCmd.Args = cobra.NoArgs

	shimCmd.Flags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	shimCmd.MarkFlagRequired("bundle")
	shimCmd.Flags().String("container-id", "", "container id")
	shimCmd.MarkFlagRequired("container-id")
	shimCmd.Flags().String("runtime", "yacr", "container runtime to use")
}

func shim(cmd *cobra.Command, args []string) error {
	cfg, err := config.NewShimConfigFromFlags(cmd.Flags())
	if err != nil {
		return err
	}

	ctx := &daemon.Context{
		PidFileName: cfg.ContainerPidFileName(),
		PidFilePerm: 0o644,
	}

	parent, err := ctx.Reborn()
	if err != nil {
		return fmt.Errorf("failed to create daemon: %w", err)
	}
	if parent != nil {
		fmt.Println(cfg.SocketAddress())
		return nil
	}
	defer ctx.Release()

	logger := logrus.WithFields(logrus.Fields{
		"id":  cfg.ContainerID(),
		"cmd": "shim",
	})

	// The daemon shim has started. We cannot log information to stdout/stderr
	// so we are going to use `logger.Fatal()` in case of an error.
	logger.Debug("started")

	// Make this daemon a subreaper so that it "adopts" orphaned descendants,
	// see: https://man7.org/linux/man-pages/man2/prctl.2.html
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		logger.WithError(err).Fatal("prctl() failed")
	}

	go createContainer(cfg, logger, cmd)

	go createHttpServer(cfg, logger)

	<-exitShim

	cfg.Destroy()

	logger.Debug("stopped")
	return nil
}

func createHttpServer(config *config.ShimConfig, logger *logrus.Entry) {
	server := http.Server{}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		switch r.Method {
		case "GET":
			sendShimState(w, config)
			return
		case "POST":
			break
		case "DELETE":
			w.Write([]byte("bye\n"))
			cancel()
			return
		default:
			msg := fmt.Sprintf("invalid method: %s\n", r.Method)
			http.Error(w, msg, http.StatusMethodNotAllowed)
			return
		}

		if config.ContainerStatus() == nil {
			msg := fmt.Sprintf("container not yet created")
			http.Error(w, msg, http.StatusNotFound)
			return
		}

		cmd := r.FormValue("cmd")
		switch cmd {
		case "start":
			state := getContainerState(w, config)
			if state == nil {
				return
			}

			if state.Status != "created" {
				msg := fmt.Sprintf("container '%s' is %s", config.ContainerID(), state.Status)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			_, err := executeRuntime(w, config, []string{"start", config.ContainerID()})
			if err != nil {
				return
			}

			sendShimState(w, config)

		case "kill":
			state := getContainerState(w, config)
			if state == nil {
				return
			}

			if state.Status != "running" {
				msg := fmt.Sprintf("container '%s' is %s", config.ContainerID(), state.Status)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			signal := "15"
			if sig := r.FormValue("signal"); sig != "" {
				signal = sig
			}

			_, err := executeRuntime(w, config, []string{"kill", config.ContainerID(), signal})
			if err != nil {
				return
			}

			sendShimState(w, config)

		case "delete":
			state := getContainerState(w, config)
			if state == nil {
				return
			}

			if state.Status != "stopped" {
				msg := fmt.Sprintf("container '%s' is %s", config.ContainerID(), state.Status)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			_, err := executeRuntime(w, config, []string{"delete", config.ContainerID()})
			if err != nil {
				return
			}

			w.WriteHeader(http.StatusNoContent)

		default:
			msg := fmt.Sprintf("invalid command '%s'", cmd)
			http.Error(w, msg, http.StatusBadRequest)
		}
	})

	http.HandleFunc("/stdout", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, config.StdoutFileName())
	})

	http.HandleFunc("/stderr", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, config.StderrFileName())
	})

	listener, err := net.Listen("unix", config.SocketAddress())
	if err != nil {
		logger.WithError(err).Fatal("failed to listen to socket")
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("serve() failed")
		}
	}()

	select {
	case <-ctx.Done():
		server.Shutdown(ctx)
	}

	close(exitShim)
}

func executeRuntime(w http.ResponseWriter, config *config.ShimConfig, runtimeArgs []string) ([]byte, error) {
	if config.Debug() {
		runtimeArgs = append(runtimeArgs, "--debug")
	}

	output, err := exec.Command(config.RuntimePath(), runtimeArgs...).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			if bytes.Contains(exitError.Stderr, []byte("not found")) {
				msg := fmt.Sprintf("container '%s' not found", config.ContainerID())
				http.Error(w, msg, http.StatusNotFound)
				return output, err
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return output, err
}

func getContainerState(w http.ResponseWriter, cfg *config.ShimConfig) *specs.State {
	output, err := executeRuntime(w, cfg, []string{"state", cfg.ContainerID()})
	if err != nil {
		return nil
	}

	var state specs.State
	if json.Unmarshal(output, &state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return &state
}

// sendShimState sends a HTTP response with the shim state, unless there is an
// error in which case the error is returned to the client.
func sendShimState(w http.ResponseWriter, cfg *config.ShimConfig) {
	state := getContainerState(w, cfg)
	if state == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"id":     cfg.ContainerID(),
		"state":  state,
		"status": cfg.ContainerStatus(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

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

	runtimeArgs := appendGlobalFlags(
		cmd,
		[]string{
			"runtime",
			"--bundle", cfg.Bundle(),
			"--container-id", cfg.ContainerID(),
			"--container-pidfile", cfg.ContainerPidFileName(),
			"--runtime", cfg.Runtime(),
		},
	)

	self, _ := os.Executable()
	process := &exec.Cmd{
		Path:   self,
		Args:   append([]string{programName}, runtimeArgs...),
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
