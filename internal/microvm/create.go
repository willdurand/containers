package microvm

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/microvm/container"
	"golang.org/x/sys/unix"

	_ "embed"
)

type CreateOpts struct {
	PidFile       string
	ConsoleSocket string
	Debug         bool
}

//go:embed init
var initBinary []byte

var charDeviceRedirected = regexp.MustCompile("char device redirected to (.+?) .+")

func Create(rootDir, containerId, bundle string, opts CreateOpts) error {
	container, err := container.New(rootDir, containerId, bundle)
	if err != nil {
		return err
	}

	if opts.PidFile == "" {
		opts.PidFile = filepath.Join(container.BaseDir, "container.pid")
	}

	// Prepare the VM root filesystem: we mainly want to install our own
	// `init(1)` executable.
	if err := os.Remove(container.InitFilePath()); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if err := os.MkdirAll(container.InitDirPath(), 0o755); err != nil {
		return err
	}
	if err := ioutil.WriteFile(container.InitFilePath(), initBinary, 0o755); err != nil {
		return err
	}

	// We use `virtiofsd` to "mount" the root filesystem in the VM.
	virtiofsd, err := exec.LookPath("virtiofsd")
	if err != nil {
		return err
	}

	virtiofsdCmd := exec.Command(
		virtiofsd,
		"--syslog",
		"--socket-path", container.VirtiofsdSocketPath(),
		"--shared-dir", container.Rootfs(),
		"--cache", "never",
		"--sandbox", "none",
	)
	// Only useful when `--syslog` isn't specified above.
	virtiofsdCmd.Stderr = os.Stderr

	logrus.WithField("command", virtiofsdCmd.String()).Debug("starting virtiofsd")
	if err := virtiofsdCmd.Start(); err != nil {
		return fmt.Errorf("virtiofsd: %w", err)
	}
	defer virtiofsdCmd.Process.Release()

	qemu, err := exec.LookPath("qemu-system-x86_64")
	if err != nil {
		return err
	}

	qemuCmd := exec.Command(qemu, container.ArgsForQEMU(opts.PidFile, opts.Debug)...)

	if opts.ConsoleSocket == "" {
		for _, p := range []string{container.PipePathIn(), container.PipePathOut()} {
			if err := unix.Mkfifo(p, 0o600); err != nil && !errors.Is(err, fs.ErrExist) {
				return err
			}
		}

		qemuCmd.Args = append(
			qemuCmd.Args,
			"-chardev", fmt.Sprintf("pipe,path=%s,id=virtiocon0", container.PipePath()),
		)
	} else {
		qemuCmd.Args = append(qemuCmd.Args, "-chardev", "pty,id=virtiocon0")
	}

	logrus.WithField("command", qemuCmd.String()).Debug("starting QEMU")
	output, err := qemuCmd.CombinedOutput()
	qemuOutput := strings.TrimSuffix(string(output), "\n")
	if err != nil {
		return fmt.Errorf("qemu: %s: %w", qemuOutput, err)
	}

	if opts.ConsoleSocket == "" {
		// If we do not have a console socket, we'll have to spawn a
		// process to redirect the microvm IOs (using the named pipes
		// created above and the host standard streams).
		self, err := os.Executable()
		if err != nil {
			return err
		}

		redirectCmd := exec.Command(self, "--root", rootDir, "redirect-stdio", containerId)
		redirectCmd.Stdin = os.Stdin
		redirectCmd.Stderr = os.Stderr
		redirectCmd.Stdout = os.Stdout

		// We need to save the container so that the `redirect-stdio`
		// command can load it.
		container.Save()

		logrus.WithField("command", redirectCmd.String()).Debug("start redirect-stdio process")
		if err := redirectCmd.Start(); err != nil {
			return err
		}
		defer redirectCmd.Process.Release()
	} else {
		// We need to retrieve the PTY file created by QEMU, which is
		// printed to stdout usually. There must be a better way to do
		// this (than parsing stdout...) but that works so... let's
		// revisit this approach later, maybe.
		matches := charDeviceRedirected.FindStringSubmatch(qemuOutput)
		if len(matches) != 2 {
			return fmt.Errorf("failed to retrieve PTY file descriptor in: %s", qemuOutput)
		}
		ptyFile := strings.TrimSpace(matches[1])

		logrus.WithField("ptyFile", ptyFile).Debug("found PTY file")

		pty, err := os.OpenFile(ptyFile, os.O_RDWR, 0o600)
		if err != nil {
			return err
		}

		// Connect to the socket in order to send the PTY file
		// descriptor.
		conn, err := net.Dial("unix", opts.ConsoleSocket)
		if err != nil {
			return err
		}
		defer conn.Close()

		uc, ok := conn.(*net.UnixConn)
		if !ok {
			return errors.New("failed to cast unix socket")
		}
		defer uc.Close()

		// Send file descriptor over socket.
		oob := unix.UnixRights(int(pty.Fd()))
		uc.WriteMsgUnix([]byte(pty.Name()), oob, nil)
	}

	data, err := ioutil.ReadFile(opts.PidFile)
	if err != nil {
		return err
	}
	pid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
	if err != nil {
		return err
	}
	container.SetPid(pid)

	return container.SaveAsCreated()
}
