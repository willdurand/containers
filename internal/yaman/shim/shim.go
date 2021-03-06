package shim

import (
	"bufio"
	"bytes"
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
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/constants"
	"github.com/willdurand/containers/internal/logs"
	"github.com/willdurand/containers/internal/yacs"
	"github.com/willdurand/containers/internal/yaman/container"
	"golang.org/x/term"
)

const (
	logFileName              = "shim.log"
	stateFileName            = "shim.json"
	slirp4netnsPidFileName   = "slirp4netns.pid"
	slirp4netnsApiSocketName = "slirp4netns.sock"
)

var executableNotFound = regexp.MustCompile("exec: .+? no such file or directory")

// ShimOpts contains the options that can be passed to a shim.
type ShimOpts struct {
	Runtime string
}

// Shim represents an instance of the `yacs` shim.
type Shim struct {
	BaseDir    string
	Container  *container.Container
	Opts       ShimOpts
	SocketPath string
	State      *yacs.YacsState
	httpClient *http.Client
}

var defaultShimOpts = ShimOpts{
	Runtime: "yacr",
}

// New creates a new shim instance for a given container.
func New(container *container.Container, opts ShimOpts) *Shim {
	shim := &Shim{
		BaseDir:   container.BaseDir,
		Container: container,
		Opts:      defaultShimOpts,
	}

	if opts.Runtime != "" {
		shim.Opts.Runtime = opts.Runtime
	}

	return shim
}

// Load attempts to load a shim configuration from disk. It returns a new shim
// instance when it succeeds or an error when there is a problem.
func Load(rootDir, id string) (*Shim, error) {
	containerDir := filepath.Join(container.GetBaseDir(rootDir), id)
	if _, err := os.Stat(containerDir); err != nil {
		return nil, fmt.Errorf("container '%s' does not exist", id)
	}

	data, err := os.ReadFile(filepath.Join(containerDir, stateFileName))
	if err != nil {
		return nil, err
	}

	shim := new(Shim)
	if err := json.Unmarshal(data, shim); err != nil {
		logrus.WithError(err).Warn("failed to load shim")
		return nil, err
	}

	if err := shim.Container.Refresh(); err != nil {
		return nil, err
	}

	return shim, nil
}

// Create starts a shim process, which will also create a container by invoking
// an OCI runtime.
func (s *Shim) Create(rootDir string) error {
	defer func() {
		if !s.Container.IsCreated() {
			emptyState := new(yacs.YacsState)
			if err := s.cleanUp(emptyState); err != nil {
				logrus.WithError(err).Info("failed to clean-up shim")
			}
		}
	}()

	if err := s.Container.Mount(); err != nil {
		return err
	}

	// Look up the path to the `yacs` shim binary.
	yacs, err := exec.LookPath("yacs")
	if err != nil {
		return err
	}

	// Save the shim's state in case we need to load it in hooks.
	if err := s.save(); err != nil {
		return err
	}

	self, err := os.Executable()
	if err != nil {
		return err
	}

	// Prepare a list of arguments for `yacs`.
	args := []string{
		// Specify the base directory so that we keep most of the files in the same
		// "container directory", which should also help when we need to clean-up
		// everything because of an error.
		"--base-dir", filepath.Join(s.BaseDir, "shim"),
		"--log", s.logFilePath(),
		// With JSON logs, we can parse the error message in case of an error.
		"--log-format", "json",
		"--bundle", s.Container.BaseDir,
		"--container-id", s.Container.ID,
		"--container-log-file", s.Container.LogFilePath,
		"--stdio-dir", s.stdioDir(),
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
		err = logs.GetBetterError(s.logFilePath(), err)
		if executableNotFound.MatchString(err.Error()) {
			return cli.ExitCodeError{Message: err.Error(), ExitCode: 127}
		}

		return err
	}

	// When `yacs` starts, it should print a unix socket path to the standard
	// output so that we can communicate with it via a HTTP API.
	s.SocketPath = strings.TrimSpace(string(data))
	s.Container.CreatedAt = time.Now()

	return s.save()
}

// GetState queries the shim to retrieve its state and returns it.
func (s *Shim) GetState() (*yacs.YacsState, error) {
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

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	state := new(yacs.YacsState)
	if err := json.Unmarshal(data, state); err != nil {
		return nil, fmt.Errorf("failed to read state: %s", data)
	}

	return state, nil
}

// Terminate stops the shim if the container is stopped, otherwise an error is
// returned.
//
// Stopping the shim is performed in two steps: (1) delete the container, and
// (2) run some clean-up tasks like unmounting the root filesystem, stopping
// slirp4netns and terminating the shim process.
//
// Once this is done, we persist the final shim state on disk so that other
// Yaman commands can read and display information until the container is
// actually deleted. This is one of the main differences with the `Destroy()`
// method: the shim state is still available.
func (s *Shim) Terminate() error {
	// We need to read the state first because we won't be able to read it once
	// the container has been deleted (by the OCI runtime).
	state, err := s.GetState()
	if err != nil {
		return err
	}

	if err := s.DeleteContainer(); err != nil {
		return err
	}

	return s.cleanUp(state)
}

// Delete deletes a container that is not running, otherwise an error will be
// returned. If the container is not running and not stopped, the shim is
// terminated first.
//
// All the container files should be deleted as a result of a call to this
// method and the container will not exist anymore.
func (s *Shim) Delete() error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	switch state.State.Status {
	case constants.StateRunning:
		return fmt.Errorf("container '%s' is %s", s.Container.ID, state.State.Status)
	case constants.StateStopped:
		break
	default:
		if err := s.cleanUp(state); err != nil {
			return err
		}
	}

	return s.Container.Delete()
}

// CopyLogs copies all the container logs stored by the shim to the provided
// writers. Note that this method does NOT use the shim's HTTP API. It reads the
// container log file directly.
func (s *Shim) CopyLogs(stdout io.Writer, stderr io.Writer, withTimestamps bool) error {
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
func (s *Shim) StartContainer() error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	if state.State.Status != constants.StateCreated {
		return fmt.Errorf("container '%s' is %s", s.Container.ID, state.State.Status)
	}

	err = s.sendCommand(url.Values{
		"cmd": []string{"start"},
	})
	if err != nil {
		if executableNotFound.MatchString(err.Error()) {
			// Remove the prefix set by `sendCommand`.
			return cli.ExitCodeError{
				Message:  strings.TrimSuffix(err.Error()[7:], "\n"),
				ExitCode: 127,
			}
		}
	}

	return err
}

// StopContainer tells the shim to stop the container by sending a SIGTERM
// signal first and a SIGKILL if the first signal didn't stop the container.
func (s *Shim) StopContainer() error {
	if err := s.sendCommand(url.Values{
		"cmd":    []string{"kill"},
		"signal": []string{"SIGTERM"},
	}); err != nil {
		return err
	}

	// Wait a second before reading the state again.
	time.Sleep(1 * time.Second)

	state, err := s.GetState()
	if err != nil {
		// After a second, it is possible that the container has exited properly
		// after SIGTEM and we ran 'container cleanup'.
		return nil
	}

	if state.State.Status != constants.StateStopped {
		logrus.WithField("id", s.Container.ID).Debug("SIGTERM failed, sending SIGKILL")

		if err := s.sendCommand(url.Values{
			"cmd":    []string{"kill"},
			"signal": []string{"SIGKILL"},
		}); err != nil {
			return err
		}

		// Give some more time to the exit command.
		time.Sleep(1 * time.Second)
	}

	return nil
}

// DeleteContainer tells the shim to delete the container.
func (s *Shim) DeleteContainer() error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	if state.State.Status != constants.StateStopped {
		return fmt.Errorf("container '%s' is %s", s.Container.ID, state.State.Status)
	}

	return s.sendCommand(url.Values{"cmd": []string{"delete"}})
}

// OpenStreams opens and returns the stdio streams of the container.
func (s *Shim) OpenStreams() (*os.File, *os.File, *os.File, error) {
	stdin, err := os.OpenFile(filepath.Join(s.stdioDir(), "0"), os.O_WRONLY, 0)
	if err != nil {
		return nil, nil, nil, err
	}

	stdout, err := os.Open(filepath.Join(s.stdioDir(), "1"))
	if err != nil {
		return nil, nil, nil, err
	}

	stderr, err := os.Open(filepath.Join(s.stdioDir(), "2"))
	if err != nil {
		return nil, nil, nil, err
	}

	return stdin, stdout, stderr, nil
}

// Attach attaches the provided Input/Output streams to the container.
func (s *Shim) Attach(attachStdin, attachStdout, attachStderr bool) error {
	stdin, stdout, stderr, err := s.OpenStreams()
	if err != nil {
		return err
	}
	defer stdin.Close()
	defer stdout.Close()
	defer stderr.Close()

	// In interactive mode, we keep `stdin` open, otherwise we close it
	// immediately and we only care about `stdout` and `stderr`.
	if attachStdin {
		go func() {
			io.Copy(stdin, os.Stdin)

			if !s.Container.Opts.Tty {
				// HACK: this isn't how we should handle EOF on stdin but there is an
				// issue with using the named pipes directly. Closing `stdin` here
				// isn't enough because the shim keeps it open (on purpose...). We need
				// "something" to close here so that the shim can close the named pipe
				// itself but sending the string below isn't what we should be doing...
				stdin.Write([]byte("\nTHIS_IS_NOT_HOW_WE_SHOULD_CLOSE_A_PIPE\n"))
			}
		}()
	} else {
		stdin.Close()
	}

	if s.Container.Opts.Tty {
		// TODO: maybe handle the case where we want to detach from the container
		// without killing it. Docker has a special key sequence for detaching a
		// container.

		// We force the current terminal to switch to "raw mode" because we don't
		// want it to mess with the PTY set up by the container itself.
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)

		go io.Copy(stdin, os.Stdin)
		// Block on the stream coming from the container so that when it exits, we
		// can also exit this command.
		io.Copy(os.Stdout, stdout)
	} else {
		// TODO: proxy all received signals to the container process and maybe add
		// an option like Docker's `--sig-proxy` one.

		var wg sync.WaitGroup
		// We copy the data from the container to the appropriate streams as long
		// as we can. When the container process exits, the shimm should close the
		// streams on its end, which should allow `copyStd()` to complete.
		if attachStdout {
			wg.Add(1)
			go copyStd(stdout, os.Stdout, &wg)
		}

		if attachStderr {
			wg.Add(1)
			go copyStd(stderr, os.Stderr, &wg)
		}

		wg.Wait()
	}

	return nil
}

// Slirp4netnsPidFilePath returns the path to the file where the slirp4netns
// process ID should be written when it is started.
func (s *Shim) Slirp4netnsPidFilePath() string {
	return filepath.Join(s.BaseDir, slirp4netnsPidFileName)
}

// Slirp4netnsApiSocketPath returns the path to the API socket used to
// communicate with the slirp4netns process.
func (s *Shim) Slirp4netnsApiSocketPath() string {
	return filepath.Join(s.BaseDir, slirp4netnsApiSocketName)
}

// Recreate stops a container if it is running and then re-create a new
// container. If the container is already stopped, we only re-create the
// container.
func (s *Shim) Recreate(rootDir string) error {
	state, err := s.GetState()
	if err != nil {
		return err
	}

	// When the container has only been created, we can early return because
	// there is no need to recreate it.
	if state.State.Status == constants.StateCreated {
		return nil
	}

	// Backup the remove option because we might have to override it if the
	// container is not stopped and we have to stop it.
	remove := s.Container.Opts.Remove

	if state.State.Status != constants.StateStopped {
		// We need to set this option to `false` unconditionally because we
		// don't want to auto-remove the container when we stop it since we want
		// to restart it.
		s.Container.Opts.Remove = false
		if err := s.save(); err != nil {
			return err
		}

		if err := s.StopContainer(); err != nil {
			return err
		}
	}

	// Reset shim state
	s.State = nil
	// Restore remove option
	s.Container.Opts.Remove = remove
	// Reset container state
	s.Container.ExitedAt = time.Time{}
	if err := s.save(); err != nil {
		return err
	}

	// At this point the container has been stopped: the shim process is gone
	// and the rootfs has been unmounted. We should create a new container and
	// its shim process.
	return s.Create(rootDir)
}

func (s *Shim) cleanUp(state *yacs.YacsState) error {
	if err := s.Container.Unmount(); err != nil {
		return err
	}

	if _, err := os.Stat(s.Slirp4netnsPidFilePath()); err == nil {
		if data, err := os.ReadFile(s.Slirp4netnsPidFilePath()); err == nil {
			if slirpPid, err := strconv.Atoi(string(bytes.TrimSpace(data))); err == nil {
				logrus.WithField("pid", slirpPid).Debug("terminating slirp4netns")

				if err := syscall.Kill(slirpPid, syscall.SIGTERM); err != nil {
					logrus.WithError(err).Debug("failed to terminate slirp4netns")
				}
			}
		}

		if err := os.Remove(s.Slirp4netnsPidFilePath()); err != nil {
			logrus.WithError(err).Debug("failed to delete slirp4netns pid file")
		}
	}

	if err := os.Remove(s.Slirp4netnsApiSocketPath()); err != nil {
		logrus.WithError(err).Debug("failed to delete slirp4netns socket file")
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

	return s.save()
}

func (s *Shim) sendCommand(values url.Values) error {
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

func (s *Shim) getHttpClient() (*http.Client, error) {
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

func (s *Shim) save() error {
	// Persist the state of the shim to disk.
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.stateFilePath(), data, 0o644)
}

func (s *Shim) logFilePath() string {
	return filepath.Join(s.BaseDir, logFileName)
}

func (s *Shim) stateFilePath() string {
	return filepath.Join(s.BaseDir, stateFileName)
}

func (s *Shim) stdioDir() string {
	return s.BaseDir
}
