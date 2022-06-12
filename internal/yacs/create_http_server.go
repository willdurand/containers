package yacs

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os/exec"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/constants"
)

// CreateHttpServer creates a HTTP server to expose an API to interact with the
// shim.
func (y *Yacs) CreateHttpServer(logger *logrus.Entry) {
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
			y.sendShimStateOrHttpError(w)
			return
		case "POST":
			y.processCommand(w, r)
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

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, y.ContainerLogFile)
	})

	listener, err := net.Listen("unix", y.SocketAddress())
	if err != nil {
		logger.WithError(err).Panic("failed to listen to socket")
	}

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logger.WithError(err).Panic("serve() failed")
		}
	}()

	<-ctx.Done()

	server.Shutdown(ctx)
	close(y.Exit)
}

func (y *Yacs) processCommand(w http.ResponseWriter, r *http.Request) {
	if y.containerStatus == nil {
		http.Error(w, "container not yet created", http.StatusNotFound)
		return
	}

	cmd := r.FormValue("cmd")
	switch cmd {
	case "start":
		state := y.getContainerStateOrHttpError(w)
		if state == nil {
			return
		}

		if state.Status != constants.StateCreated {
			msg := fmt.Sprintf("container '%s' is %s", y.ContainerID, state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		_, err := y.executeRuntimeOrHttpError(w, []string{"start", y.ContainerID})
		if err != nil {
			return
		}

		y.sendShimStateOrHttpError(w)

	case "kill":
		state := y.getContainerStateOrHttpError(w)
		if state == nil {
			return
		}

		if state.Status != constants.StateRunning {
			msg := fmt.Sprintf("container '%s' is %s", y.ContainerID, state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		_, err := y.executeRuntimeOrHttpError(w, []string{"kill", y.ContainerID, r.FormValue("signal")})
		if err != nil {
			return
		}

		y.sendShimStateOrHttpError(w)

	case "delete":
		state := y.getContainerStateOrHttpError(w)
		if state == nil {
			return
		}

		if state.Status != constants.StateStopped {
			msg := fmt.Sprintf("container '%s' is %s", y.ContainerID, state.Status)
			http.Error(w, msg, http.StatusBadRequest)
			return
		}

		_, err := y.executeRuntimeOrHttpError(w, []string{"delete", y.ContainerID})
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
func (y *Yacs) executeRuntimeOrHttpError(w http.ResponseWriter, runtimeArgs []string) ([]byte, error) {
	output, err := exec.Command(
		y.runtimePath,
		// Add default runtime args.
		append(y.runtimeArgs(), runtimeArgs...)...,
	).Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			// HACK: we should probably not parse the error message like that...
			// Note that this should work with `runc` too, though.
			if bytes.Contains(exitError.Stderr, []byte("does not exist")) {
				msg := fmt.Sprintf("container '%s' does not exist", y.ContainerID)
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
func (y *Yacs) getContainerStateOrHttpError(w http.ResponseWriter) *runtimespec.State {
	output, err := y.executeRuntimeOrHttpError(w, []string{"state", y.ContainerID})
	if err != nil {
		return nil
	}

	var state runtimespec.State
	if json.Unmarshal(output, &state); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	return &state
}

// sendShimStateOrHttpError sends a HTTP response with the shim state, unless
// there is an error in which case the error is returned to the client.
func (y *Yacs) sendShimStateOrHttpError(w http.ResponseWriter) {
	state := y.getContainerStateOrHttpError(w)
	if state == nil {
		return
	}

	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"id":      y.ContainerID,
		"state":   state,
		"runtime": y.runtime,
		"status":  y.containerStatus,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
