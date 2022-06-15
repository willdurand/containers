package shim

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/yacs"
	"github.com/willdurand/containers/internal/yaman/container"
)

// Yacs represents an instance of the `yacs` shim.
type Yacs struct {
	BaseShim
	SocketPath string
	State      *YacsState
	httpClient *http.Client
}

// YacsState represents the state of the `yacs` shim.
type YacsState struct {
	State  runtimespec.State
	Status *yacs.ContainerStatus
}

var defaultYacsOpts = ShimOpts{
	Runtime: "yacr",
}

// New creates a new shim instance for a given container.
func New(container *container.Container, opts ShimOpts) *Yacs {
	shim := &Yacs{
		BaseShim: BaseShim{
			Container: container,
			Opts:      defaultYacsOpts,
		},
	}

	if opts.Runtime != "" {
		shim.Opts.Runtime = opts.Runtime
	}

	return shim
}

// Load attempts to load a shim configuration from disk. It returns a new shim
// instance when it succeeds or an error when there is a problem.
func Load(rootDir, id string) (*Yacs, error) {
	containerDir := filepath.Join(container.GetBaseDir(rootDir), id)
	if _, err := os.Stat(containerDir); err != nil {
		return nil, fmt.Errorf("container '%s' does not exist", id)
	}

	data, err := os.ReadFile(filepath.Join(containerDir, stateFileName))
	if err != nil {
		return nil, err
	}

	shim := new(Yacs)
	if err := json.Unmarshal(data, shim); err != nil {
		logrus.WithError(err).Warn("failed to load shim")
		return nil, err
	}

	return shim, nil
}

// GetState queries the shim to retrieve its state and returns it.
func (s *Yacs) GetState() (*YacsState, error) {
	c, err := s.getHttpClient()
	if err != nil {
		return nil, err
	}

	resp, err := c.Get("http://shim/")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	state := new(YacsState)
	if err := json.NewDecoder(resp.Body).Decode(state); err != nil {
		return nil, err
	}

	return state, nil
}

// Destroy destroys a stopped container, otherwise an error will be returned.
func (s *Yacs) Destroy() error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	if state.State.Status != constants.StateStopped {
		return fmt.Errorf("container '%s' is %s", s.ID(), state.State.Status)
	}

	return s.Container.Destroy()
}

// CopyLogs copies all the container logs returned by the shim to the provided
// writers.
func (s *Yacs) CopyLogs(stdout io.Writer, stderr io.Writer, withTimestamps bool) error {
	file, err := os.Open(s.Container.LogFilePath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}

		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var l map[string]string
		if err := json.Unmarshal(scanner.Bytes(), &l); err != nil {
			return err
		}

		data := append([]byte(l["m"]), '\n')
		if withTimestamps {
			if t, err := time.Parse(time.RFC3339, l["t"]); err == nil {
				data = append(
					// TODO: I wanted to use time.RFC3339Nano but the length isn't fixed
					// and that breaks the alignement when rendered.
					[]byte(t.Local().Format(time.RFC3339)),
					append([]byte{' ', '-', ' '}, data...)...,
				)
			}
		}

		if l["s"] == "stderr" {
			stderr.Write(data)
		} else {
			stdout.Write(data)
		}
	}

	return nil
}

func (s *Yacs) getHttpClient() (*http.Client, error) {
	if s.SocketPath == "" {
		return nil, fmt.Errorf("container '%s' is not running", s.Container.ID)
	}

	if s.httpClient == nil {
		s.httpClient = &http.Client{
			Transport: &http.Transport{
				DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
					return net.Dial("unix", s.SocketPath)
				},
			},
		}
	}

	return s.httpClient, nil
}
