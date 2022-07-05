package main

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
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/cli"
	"github.com/willdurand/containers/internal/microvm/container"
	"golang.org/x/sys/unix"

	_ "embed"
)

//go:embed init
var initBinary []byte

func init() {
	createCmd := &cobra.Command{
		Use:   "create <id>",
		Short: "Create a container",
		Run: cli.HandleErrors(func(cmd *cobra.Command, args []string) error {
			rootDir, _ := cmd.Flags().GetString("root")
			bundle, _ := cmd.Flags().GetString("bundle")
			pidFile, _ := cmd.Flags().GetString("pid-file")
			consoleSocket, _ := cmd.Flags().GetString("console-socket")
			debug, _ := cmd.Flags().GetBool("debug")

			container, err := container.New(rootDir, args[0], bundle)
			if err != nil {
				return err
			}

			if pidFile == "" {
				pidFile = filepath.Join(container.BaseDir, "container.pid")
			}

			// Prepare the VM root filesystem: we mainly want to install our own
			// `init(1)` process.
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
				// HACK: ugh!
				"sudo",
				virtiofsd,
				"--syslog",
				"--socket-path", container.VirtiofsdSocketPath(),
				"--shared-dir", container.Rootfs(),
				"--socket-group", "gitpod",
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

			time.Sleep(10 * time.Millisecond)

			qemu, err := exec.LookPath("qemu-system-x86_64")
			if err != nil {
				return err
			}

			qemuCmd := exec.Command(
				qemu,
				"-M", "microvm",
				"-m", "512m",
				"-no-acpi", "-no-reboot", "-no-user-config", "-nodefaults", "-display", "none",
				"-device", "virtio-serial-device",
				"-device", "virtconsole,chardev=virtiocon0",
				"-chardev", fmt.Sprintf("socket,id=virtiofs0,path=%s", container.VirtiofsdSocketPath()),
				"-device", "vhost-user-fs-device,queue-size=1024,chardev=virtiofs0,tag=/dev/root",
				"-kernel", "/workspace/containers/microvm/build/vmlinux",
				"-object", fmt.Sprintf("memory-backend-file,id=mem,size=%s,mem-path=%s,share=on", "512m", filepath.Join(container.BaseDir, "shm")),
				"-numa", "node,memdev=mem",
				"-pidfile", pidFile, "-daemonize",
				"-append", container.AppendLine(debug),
			)

			if consoleSocket == "" {
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

			if consoleSocket == "" {
				// If we do not have a console socket, we'll have to spawn a
				// process to redirect the microvm IOs (using the named pipes
				// created above and the host standard streams).
				self, err := os.Executable()
				if err != nil {
					return err
				}

				redirectCmd := exec.Command(self, "--root", rootDir, "redirect-stdio", args[0])
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
				charDeviceRedirected := regexp.MustCompile("char device redirected to (.+?) .+")
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
				conn, err := net.Dial("unix", consoleSocket)
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

			data, err := ioutil.ReadFile(pidFile)
			if err != nil {
				return err
			}
			pid, err := strconv.Atoi(string(bytes.TrimSpace(data)))
			if err != nil {
				return err
			}
			container.SetPid(pid)

			return container.SaveAsCreated()
		}),
		Args: cobra.ExactArgs(1),
	}
	createCmd.PersistentFlags().StringP("bundle", "b", "", "path to the root of the bundle directory")
	createCmd.MarkFlagRequired("bundle")
	createCmd.Flags().String("pid-file", "", "specify the file to write the process id to")
	createCmd.Flags().String("console-socket", "", "console unix socket used to pass a PTY descriptor")
	rootCmd.AddCommand(createCmd)
}
