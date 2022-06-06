package shim

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

// Start starts a shim process, which will create a container by invoking an
// OCI runtime.
func (s *Yacs) Start(rootDir string) error {
	// Look up the path to the `yacs` shim binary.
	yacs, err := exec.LookPath("yacs")
	if err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	// Prepare a list of arguments for `yacs`.
	args := []string{
		"--bundle", s.Container.BaseDir,
		"--container-id", s.Container.ID,
		"--container-log-file", s.Container.LogFilePath,
		"--stdio-dir", s.Container.BaseDir,
		"--runtime", s.Opts.Runtime,
		"--exit-command", self,
		"--exit-command-arg", "--root",
		"--exit-command-arg", rootDir,
		"--exit-command-arg", "container",
		"--exit-command-arg", "cleanup",
		"--exit-command-arg", s.Container.ID,
	}
	if logrus.IsLevelEnabled(logrus.DebugLevel) {
		args = append(args, []string{
			// For the exit command...
			"--exit-command-arg", "--debug",
			// ...and for the shim.
			"--debug",
		}...)
	}

	// Create the command to execute to start the shim.
	shimCmd := exec.Command(yacs, args...)

	logrus.WithFields(logrus.Fields{
		"command": shimCmd.String(),
	}).Debug("start shim")

	data, err := shimCmd.Output()
	if err != nil {
		logrus.WithError(err).Error("failed to start shim")
		return err
	}

	// When `yacs` starts, it should print a unix socket path to the standard
	// output so that we can communicate with it via a HTTP API.
	s.SocketPath = strings.TrimSpace(string(data))

	// In theory, `yacs` should only print the socket path when it has fully
	// initialized itself but that does not seem to work very well so let's wait
	// a bit to make sure the shim is ready...
	for i := 0; i < 10; i++ {
		time.Sleep(50 * time.Millisecond)

		if _, err := s.GetState(); err == nil {
			s.Container.StartedAt = time.Now()
			break
		}
	}

	if !s.Container.IsStarted() {
		return fmt.Errorf("failed to start container")
	}

	// Persist the state of the shim to disk.
	data, err = json.Marshal(s)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.stateFilePath(), data, 0o644); err != nil {
		return err
	}

	return nil
}

// GetState queries the shim to retrieve its state and returns it.
func (s *Yacs) GetState() (*YacsState, error) {
	// When a shim is terminated, the `State` property should be non-nil and
	// that's what we return instead of attempting to communicate with the no
	// longer existing shim.
	if s.State != nil {
		return s.State, nil
	}

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

// Terminates stops the shim if the container is stopped, otherwise an error is
// returned.
//
// Stopping the shim is performed in multiple steps: (1) delete the container,
// (2) clean-up the container (e.g., unmount rootfs), (3) terminate the shim.
// Once this is done, we persist the final shim state on disk so that other
// Yaman commands can read and display information until the container is
// actually deleted.
func (s *Yacs) Terminate() error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	if state.State.Status != constants.StateStopped {
		return fmt.Errorf("container '%s' is %s", s.Container.ID, state.State.Status)
	}

	if err := s.sendCommand(url.Values{"cmd": []string{"delete"}}); err != nil {
		return err
	}

	if err := s.Container.CleanUp(); err != nil {
		return err
	}

	// Terminate the shim process by sending a DELETE request.
	req, err := http.NewRequest(http.MethodDelete, "http://shim/", nil)
	if err != nil {
		return err
	}

	c, err := s.getHttpClient()
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Let's persist a copy of the shim state (before it got terminated) on disk.
	s.State = state
	s.SocketPath = ""
	s.Container.ExitedAt = time.Now()

	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(s.stateFilePath(), data, 0o644); err != nil {
		return err
	}

	return nil
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

// StartContainer tells the shim to start a container that was previously
// created.
func (s *Yacs) StartContainer() error {
	return s.sendCommand(url.Values{
		"cmd": []string{"start"},
	})
}

// StopContainer tells the shim to stop the container by sending a SIGTERM
// signal first and a SIGKILL if the first signal didn't stop the container.
func (s *Yacs) StopContainer() error {
	if err := s.sendCommand(url.Values{
		"cmd":    []string{"kill"},
		"signal": []string{"SIGTERM"},
	}); err != nil {
		return err
	}

	state, err := s.GetState()
	if err != nil {
		return err
	}

	if state.State.Status != constants.StateStopped {
		logrus.WithField("id", s.Container.ID).Debug("SIGTERM failed, sending SIGKILL")

		if err := s.sendCommand(url.Values{
			"cmd":    []string{"kill"},
			"signal": []string{"SIGKILL"},
		}); err != nil {
			return err
		}
	}

	return nil
}

// OpenStreams opens and returns the stdio streams of the container.
func (s *Yacs) OpenStreams() (*os.File, *os.File, *os.File, error) {
	stdin, err := os.OpenFile(filepath.Join(s.Container.BaseDir, "0"), os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := os.Open(filepath.Join(s.Container.BaseDir, "1"))
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := os.Open(filepath.Join(s.Container.BaseDir, "2"))
	if err != nil {
		return nil, nil, nil, err
	}

	return stdin, stdout, stderr, nil
}

func (s *Yacs) sendCommand(values url.Values) error {
	c, err := s.getHttpClient()
	if err != nil {
		return err
	}

	resp, err := c.PostForm("http://shim/", values)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 300 {
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return fmt.Errorf("%s: %s", values.Get("cmd"), data)
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
