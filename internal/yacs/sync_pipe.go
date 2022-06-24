package yacs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// syncPipeName is the name of the named pipe used by both the parent and child
// process to communicate when Yacs is executed. We need this pipe because we
// "fork" in order to spawn a daemon process but we don't want the parent
// process to exit too early. In fact, the parent process should wait until the
// child (daemon) process is fully initialized.
const syncPipeName = "sync-pipe"

// maybeMkfifo creates a new FIFO unless it already exists. In most cases, this
// function should return `nil` unless there is an actual error.
func (y *Yacs) maybeMkfifo() error {
	if err := unix.Mkfifo(y.syncPipePath(), 0o600); err != nil && !errors.Is(err, fs.ErrExist) {
		return err
	}

	return nil
}

// createSyncPipe creates the sync pipe and opens it in "write only" mode. The
// child (daemon) process should create this pipe.
func (y *Yacs) createSyncPipe() (*os.File, error) {
	if err := y.maybeMkfifo(); err != nil {
		return nil, err
	}

	return os.OpenFile(y.syncPipePath(), syscall.O_CREAT|syscall.O_WRONLY|syscall.O_CLOEXEC, 0)
}

// openSyncPipe opens the named pipe. This should be called by the parent
// process and this call is blocking.
func (y *Yacs) openSyncPipe() (*os.File, error) {
	if err := y.maybeMkfifo(); err != nil {
		return nil, err
	}

	return os.Open(y.syncPipePath())
}

// syncPipePath returns the path to the sync (named) pipe.
func (y *Yacs) syncPipePath() string {
	return filepath.Join(y.baseDir, syncPipeName)
}
