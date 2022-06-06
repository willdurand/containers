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
			w.Write([]byte("BYE\n"))
			cancel()
			return
		default:
			msg := fmt.Sprintf("invalid method: %s\n", r.Method)
			http.Error(w, msg, http.StatusMethodNotAllowed)
			return
		}

		if config.ContainerStatus() == nil {
			http.Error(w, "container not yet created", http.StatusNotFound)
			return
		}

		cmd := r.FormValue("cmd")
		switch cmd {
		case "start":
			state := getContainerState(w, config)
			if state == nil {
				return
			}

			if state.Status != constants.StateCreated {
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

			if state.Status != constants.StateRunning {
				msg := fmt.Sprintf("container '%s' is %s", config.ContainerID(), state.Status)
				http.Error(w, msg, http.StatusBadRequest)
				return
			}

			// TODO: handle string values and maybe use constants...
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

			if state.Status != constants.StateStopped {
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

	<-ctx.Done()

	server.Shutdown(ctx)
	close(exitShim)
}

func executeRuntime(w http.ResponseWriter, cfg *config.ShimConfig, runtimeArgs []string) ([]byte, error) {
	output, err := exec.Command(
		cfg.RuntimePath(),
		// Add default runtime args.
		append(cfg.RuntimeArgs(), runtimeArgs...)...,
	).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// HACK: we should probably not parse the error message like that...
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
		"id":      cfg.ContainerID(),
		"state":   state,
		"runtime": cfg.Runtime(),
		"status":  cfg.ContainerStatus(),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
