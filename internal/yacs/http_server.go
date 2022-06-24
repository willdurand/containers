package yacs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/constants"
)

const shimSocketName = "shim.sock"

var (
	ErrNotCreated = errors.New("container is not created")
	ErrNotRunning = errors.New("container is not running")
	ErrNotStopped = errors.New("container is not stopped")
)

// YacsState represents the "public" state of the shim.
type YacsState struct {
	ID      string
	Runtime string
	State   runtimespec.State
	Status  *ContainerStatus
}

// createHttpServer creates a HTTP server to expose an API to interact with the
// shim.
func (y *Yacs) createHttpServer() {
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

		case "POST":
			y.processCommand(w, r)

		case "DELETE":
			w.Write([]byte("BYE\n"))
			cancel()

		default:
			msg := fmt.Sprintf("invalid method: '%s'", r.Method)
			http.Error(w, msg, http.StatusMethodNotAllowed)

		}
	})

	http.HandleFunc("/logs", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, y.containerLogFilePath)
	})

	listener, err := net.Listen("unix", y.SocketPath())
	if err != nil {
		y.httpServerReady <- fmt.Errorf("listen: %w", err)
		return
	}

	// At this point, we can tell the parent that we are ready to accept
	// connections. The parent will print the socket address and exit.
	y.httpServerReady <- nil

	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Error("serve() failed")
		}
	}()

	<-ctx.Done()
	server.Shutdown(ctx)

	logrus.Debug("stopped http server")

	y.terminate()
}

// SocketPath returns the path to the unix socket used to communicate with the
// shim.
func (y *Yacs) SocketPath() string {
	return filepath.Join(y.baseDir, shimSocketName)
}

// processCommand processes an API command. If the command is valid, the OCI
// runtime is usually called and the state of the shim is returned. When
// something goes wrong, an error is returned instead.
func (y *Yacs) processCommand(w http.ResponseWriter, r *http.Request) {
	state, err := y.State()
	if err != nil {
		writeHttpError(w, err)
		return
	}

	cmd := r.FormValue("cmd")
	switch cmd {
	case "start":
		if state.Status != constants.StateCreated {
			writeHttpError(w, ErrNotCreated)
			return
		}

		if err := y.Start(); err != nil {
			writeHttpError(w, err)
			return
		}

	case "kill":
		if state.Status != constants.StateRunning {
			writeHttpError(w, ErrNotRunning)
			return
		}

		if err := y.Kill(r.FormValue("signal")); err != nil {
			writeHttpError(w, err)
			return
		}

	case "delete":
		if state.Status != constants.StateStopped {
			writeHttpError(w, ErrNotStopped)
			return
		}

		if err := y.Delete(false); err != nil {
			writeHttpError(w, err)
			return
		}

		// We cannot return the state anymore given we just deleted the
		// container.
		w.WriteHeader(http.StatusNoContent)
		return

	default:
		msg := fmt.Sprintf("invalid command '%s'", cmd)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	y.sendShimStateOrHttpError(w)
}

// sendShimStateOrHttpError sends a HTTP response with the shim state, unless
// there is an error in which case the error is returned to the client.
func (y *Yacs) sendShimStateOrHttpError(w http.ResponseWriter) {
	state, err := y.State()
	if err != nil {
		writeHttpError(w, err)
		return
	}

	if err := json.NewEncoder(w).Encode(YacsState{
		ID:      y.containerID,
		Runtime: y.runtime,
		State:   *state,
		Status:  y.containerStatus,
	}); err != nil {
		writeHttpError(w, err)
	}
}

func writeHttpError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	if errors.Is(err, ErrContainerNotExist) {
		status = http.StatusNotFound
	} else if errors.Is(err, ErrNotRunning) || errors.Is(err, ErrNotStopped) || errors.Is(err, ErrNotCreated) {
		status = http.StatusBadRequest
	}

	http.Error(w, err.Error(), status)
}
