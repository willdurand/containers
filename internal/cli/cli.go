// Package cli contains the common features used to build CLI applications.
package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/internal/version"
)

// NewRootCommand creates a new root (base) command for a program. The caller is
// responsible for assigning other properties when needed.
func NewRootCommand(programName, shortDescription string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:     programName,
		Short:   shortDescription,
		Version: version.Version(),
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd:   false,
			HiddenDefaultCmd:    true,
			DisableDescriptions: true,
		},
		PersistentPreRunE: makeRootPreRunE(programName),
	}

	rootCmd.PersistentFlags().String("root", getDefaultRootDir(programName), "root directory")
	rootCmd.PersistentFlags().String("log", "", `path to the log file (default "/dev/stderr")`)
	rootCmd.PersistentFlags().String("log-level", "warn", "set the logging level")
	rootCmd.PersistentFlags().String("log-format", "", `set the loging format (default "text")`)
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")

	return rootCmd
}

func Execute(cmd *cobra.Command) {
	if err := cmd.Execute(); err != nil {
		exit(err)
	}
}

func HandleErrors(f func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		if err := f(cmd, args); err != nil {
			logrus.Error(err)

			if !logToStderr() {
				PrintUserError(err)
			}

			exit(err)
		}
	}
}

func PrintUserError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
}

func exit(err error) {
	if exitCodeError, ok := err.(ExitCodeError); ok {
		os.Exit(exitCodeError.ExitCode)
	}

	os.Exit(125)
}

// logToStderr returns true when the logger is configured to write to stderr,
// false otherwise.
func logToStderr() bool {
	l, ok := logrus.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

// makeRootPreRunE creates a `PersistentPreRunE()` function that should be used
// on root commands to configure the logger and the program's root directory.
func makeRootPreRunE(programName string) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		rootDir, _ := cmd.Flags().GetString("root")
		if err := makeRootDir(rootDir); err != nil {
			return err
		}

		switch logFormat, _ := cmd.Flags().GetString("log-format"); logFormat {
		case "", "text":
			// do nothing
		case "json":
			logrus.SetFormatter(&logrus.JSONFormatter{})
		default:
			return fmt.Errorf("unsupported log format '%s'", logFormat)
		}

		if logFile, _ := cmd.Flags().GetString("log"); logFile != "" {
			out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}

			logrus.SetOutput(out)
		}

		logLevel, _ := cmd.Flags().GetString("log-level")
		level, err := logrus.ParseLevel(logLevel)
		if err != nil {
			level = logrus.WarnLevel
		}
		logrus.SetLevel(level)

		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			out, err := os.OpenFile(filepath.Join(rootDir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}

			logrus.SetLevel(logrus.DebugLevel)

			logrus.AddHook(&writer.Hook{
				Writer:    out,
				LogLevels: logrus.AllLevels,
			})

			logrus.WithFields(logrus.Fields{
				"program": programName,
				"args":    os.Args,
			}).Debug("invoking command")
		}

		return nil
	}
}

func makeRootDir(rootDir string) error {
	if _, err := os.Stat(rootDir); errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(rootDir, 0o700); err != nil {
			return err
		}

		xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
		if xdgRuntimeDir != "" && strings.HasPrefix(rootDir, xdgRuntimeDir) {
			// $XDG_RUNTIME_DIR defines the base directory relative to which
			// user-specific non-essential runtime files and other file objects
			// (such as sockets, named pipes, ...) should be stored. The
			// directory MUST be owned by the user, and he MUST be the only one
			// having read and write access to it. Its Unix access mode MUST be
			// 0700. [...] Files in this directory MAY be subjected to periodic
			// clean-up. To ensure that your files are not removed, they should
			// have their access time timestamp modified at least once every 6
			// hours of monotonic time or the 'sticky' bit should be set on the
			// file.
			err := os.Chmod(rootDir, os.FileMode(0o700)|os.ModeSticky)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func getDefaultRootDir(programName string) string {
	rootDir := filepath.Join("/run", programName)
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")

	if xdgRuntimeDir != "" && os.Getuid() != 0 {
		rootDir = filepath.Join(xdgRuntimeDir, programName)
	}

	envVar := fmt.Sprintf("%s_ROOT_DIR", strings.ToUpper(programName))
	if dir := os.Getenv(envVar); dir != "" {
		rootDir = dir
	}

	return rootDir
}
