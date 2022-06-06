package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/constants"
	"github.com/willdurand/containers/yacs/config"
)

// createHttpServer creates a HTTP server to expose an API to interact with the
// shim.
func createHttpServer(cfg *config.ShimConfig, logger *logrus.Entry) {
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
			sendShimStateOrHttpError(w, cfg)
			return
		case "POST":
			processCommand(w, r, cfg)
			return
		case "DELETE":
			// Shutdown the shim.
			w.Write([]byte("BYE\n"))
			cancel()
			return
		default:
			msg := fmt.Sprintf("invalid method: %s\n", r.Method)
			http.Error(w, msg, http.StatusMethodNotAllowed)
			return
		}
	})

	http.HandleFunc("/stdout", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, cfg.StdoutFileName())
	})

	http.HandleFunc("/stderr", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, cfg.StderrFileName())
	})

	listener, err := net.Listen("unix", cfg.SocketAddress())
	if err != nil {
		logger.WithError(err).Fatal("failed to listen to socket")
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Fatal("serve() failed")
		}
	}()

	<-ctx.Done()

	server.Shutdown(ctx)
	close(exitShim)
}

func processCommand(w http.ResponseWriter, r *http.Request, cfg *config.ShimConfig) {
	if cfg.ContainerStatus() == nil {
		http.Error(w, "container not yet created", http.StatusNotFound)
		return
	}

	cmd := r.FormValue("cmd")
	switch cmd {
	case "start":
		state := getContainerStateOrHttpError(w, cfg)
		if state == nil {
			return
		}

		if state.Status != constants.StateCreated {
			msg := fmt.Sprintf("container '%s' is %s", cfg.ContainerID(), state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		_, err := executeRuntimeOrHttpError(w, cfg, []string{"start", cfg.ContainerID()})
		if err != nil {
			return
		}

		sendShimStateOrHttpError(w, cfg)

	case "kill":
		state := getContainerStateOrHttpError(w, cfg)
		if state == nil {
			return
		}

		if state.Status != constants.StateRunning {
			msg := fmt.Sprintf("container '%s' is %s", cfg.ContainerID(), state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		// TODO: handle string values and maybe use constants...
		signal := "15"
		if sig := r.FormValue("signal"); sig != "" {
			signal = sig
		}

		_, err := executeRuntimeOrHttpError(w, cfg, []string{"kill", cfg.ContainerID(), signal})
		if err != nil {
			return
		}

		sendShimStateOrHttpError(w, cfg)

	case "delete":
		state := getContainerStateOrHttpError(w, cfg)
		if state == nil {
			return
		}

		if state.Status != constants.StateStopped {
			msg := fmt.Sprintf("container '%s' is %s", cfg.ContainerID(), state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		_, err := executeRuntimeOrHttpError(w, cfg, []string{"delete", cfg.ContainerID()})
		if err != nil {
			return
		}

		w.WriteHeader(http.StatusNoContent)

	default:
		msg := fmt.Sprintf("invalid command '%s'", cmd)
		http.Error(w, msg, http.StatusBadRequest)
	}
}

// executeRuntimeOrHttpError runs the OCI runtime with the command and flags
// passed in the `runtimeArgs` parameter. When something goes wrong, an HTTP
// error is written to the response write `w` and the error is returned to the
// caller.
func executeRuntimeOrHttpError(w http.ResponseWriter, cfg *config.ShimConfig, runtimeArgs []string) ([]byte, error) {
	output, err := exec.Command(
		cfg.RuntimePath(),
		// Add default runtime args.
		append(cfg.RuntimeArgs(), runtimeArgs...)...,
	).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// HACK: we should probably not parse the error message like that...
			// Note that this should work with `runc` too, though.
			if bytes.Contains(exitError.Stderr, []byte("does not exist")) {
				msg := fmt.Sprintf("container '%s' does not exist", cfg.ContainerID())
				http.Error(w, msg, http.StatusNotFound)
				return output, err
			}
		}

		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	return output, err
}

// getContainerStateOrHttpError returns the container state to the caller unless
// an error occurs, in which case an HTTP error is written to the response
// writer `w` and `nil` is returned.
//
// The container state is read from the OCI runtime (with the `state` command).
func getContainerStateOrHttpError(w http.ResponseWriter, cfg *config.ShimConfig) *specs.State {
	output, err := executeRuntimeOrHttpError(w, cfg, []string{"state", cfg.ContainerID()})
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

// sendShimStateOrHttpError sends a HTTP response with the shim state, unless
// there is an error in which case the error is returned to the client.
func sendShimStateOrHttpError(w http.ResponseWriter, cfg *config.ShimConfig) {
	state := getContainerStateOrHttpError(w, cfg)
	if state == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      cfg.ContainerID(),
		"state":   state,
		"runtime": cfg.Runtime(),
		"status":  cfg.ContainerStatus(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
