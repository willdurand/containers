package yacs

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"
	"github.com/willdurand/containers/internal/runtime"
	"golang.org/x/sys/unix"
)

const (
	containerLogFileName = "container.log"
	shimPidFileName      = "shim.pid"
)

// Yacs is a container shim.
type Yacs struct {
	baseDir              string
	bundleDir            string
	containerExited      chan interface{}
	containerLogFilePath string
	containerID          string
	containerReady       chan error
	containerSpec        runtimespec.Spec
	containerStatus      *ContainerStatus
	exitCommand          string
	exitCommandArgs      []string
	httpServerReady      chan error
	runtime              string
	runtimePath          string
	stdioDir             string
}

// NewShimFromFlags creates a new shim from a set of (command) flags. This
// function also verifies that required flags have non-empty valuey.
func NewShimFromFlags(flags *pflag.FlagSet) (*Yacs, error) {
	for _, param := range []string{
		"bundle",
		"container-id",
		"runtime",
	} {
		if v, err := flags.GetString(param); err != nil || v == "" {
			return nil, fmt.Errorf("missing or invalid value for '--%s'", param)
		}
	}

	bundleDir, _ := flags.GetString("bundle")
	spec, err := runtime.LoadSpec(bundleDir)
	if err != nil {
		return nil, err
	}

	containerId, _ := flags.GetString("container-id")
	containerLogFile, _ := flags.GetString("container-log-file")
	exitCommand, _ := flags.GetString("exit-command")
	exitCommandArgs, _ := flags.GetStringArray("exit-command-arg")
	runtime, _ := flags.GetString("runtime")
	stdioDir, _ := flags.GetString("stdio-dir")

	rootDir, _ := flags.GetString("root")
	baseDir := filepath.Join(rootDir, containerId)
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create container directory: %w", err)
	}

	if stdioDir == "" {
		stdioDir = baseDir
	}

	runtimePath, err := exec.LookPath(runtime)
	if err != nil {
		return nil, fmt.Errorf("runtime '%s' not found", runtime)
	}

	if containerLogFile == "" {
		containerLogFile = filepath.Join(baseDir, containerLogFileName)
	}

	return &Yacs{
		containerID:          containerId,
		containerLogFilePath: containerLogFile,
		baseDir:              baseDir,
		bundleDir:            bundleDir,
		containerExited:      make(chan interface{}),
		containerReady:       make(chan error),
		containerSpec:        spec,
		containerStatus:      nil,
		exitCommand:          exitCommand,
		exitCommandArgs:      exitCommandArgs,
		httpServerReady:      make(chan error),
		runtime:              runtime,
		runtimePath:          runtimePath,
		stdioDir:             stdioDir,
	}, nil
}

// Run starts the Yacs daemon. It creates a container and then the HTTP API.
//
// When everything is initialized, a message is written to the sync pipe so that
// the "parent" process can exit. Errors are also reported to the parent via the
// sync pipe.
//
// Assuming the initialization was successful, the `Run` method waits for the
// termination of the container process.
func (y *Yacs) Run() error {
	logrus.Info("the yacs daemon has started")

	// Make this daemon a subreaper so that it "adopts" orphaned descendants,
	// see: https://man7.org/linux/man-pages/man2/prctl.2.html
	if err := unix.Prctl(unix.PR_SET_CHILD_SUBREAPER, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("prctl: %w", err)
	}

	// Call the OCI runtime to create the container.
	go y.createContainer()

	syncPipe, err := y.createSyncPipe()
	if err != nil {
		return fmt.Errorf("sync pipe: %w", err)
	}
	defer syncPipe.Close()

	err = <-y.containerReady
	if err != nil {
		logrus.WithError(err).Error("failed to create container")
		syncPipe.WriteString(err.Error())
		return err
	}

	// When the container has been created, we can set up the HTTP API to be
	// able to interact with the shim and control the container.
	go y.createHttpServer()

	err = <-y.httpServerReady
	if err != nil {
		logrus.WithError(err).Error("failed to create http server")
		syncPipe.WriteString(err.Error())
		return err
	}

	// Notify the "parent" process that the initialization has completed
	// successfully.
	_, err = syncPipe.WriteString("OK")
	if err != nil {
		return err
	}

	logrus.Debug("shim successfully started")
	syncPipe.Close()

	<-y.containerExited
	return nil
}

// Err returns an error when the `Run` method has failed.
//
// This method should be used by the "parent" process. It reads data from the
// sync pipe and transforms it in an error unless the "child" process wrote a
// "OK" message.
func (y *Yacs) Err() error {
	syncPipe, err := y.openSyncPipe()
	if err != nil {
		return fmt.Errorf("open sync pipe: %w", err)
	}

	data, err := ioutil.ReadAll(syncPipe)
	if err == nil {
		if !bytes.Equal(data, []byte("OK")) {
			return errors.New(string(data))
		}
	}

	return err
}

// terminate is called when Yacs should be terminated. It will send a SIGKILL to
// the container first if it is still alive. Then, it returns the exit command
// if provided, and delete the container using the OCI runtime. After that, the
// files created by the shim are also deleted.
func (y *Yacs) terminate() {
	logrus.Debug("cleaning up before exiting")

	if err := syscall.Kill(y.containerStatus.PID, 0); err == nil {
		logrus.Debug("container still alive, sending SIGKILL")
		if err := y.Sigkill(); err != nil {
			logrus.WithError(err).Error("failed to kill container")
		}
	}

	if err := y.Delete(true); err != nil {
		logrus.WithError(err).Error("failed to force delete container")
	}

	if err := os.RemoveAll(y.baseDir); err != nil {
		logrus.WithError(err).Warn("failed to remove base directory")
	}

	close(y.containerExited)
}

// PidFilePath returns the path to the file that contains the PID of the shim.
func (y *Yacs) PidFilePath() string {
	return filepath.Join(y.baseDir, shimPidFileName)
}
