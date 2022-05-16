package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/writer"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacr/version"
)

const (
	programName string = "yacr"
)

// rootCmd represents the base command when called without any subcommands.
var rootCmd = &cobra.Command{
	Use:     programName,
	Short:   "Yet another (unsafe) container runtime",
	Version: version.Version(),
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
	SilenceErrors: true,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		switch logFormat, _ := cmd.Flags().GetString("log-format"); logFormat {
		case "text":
			// do nothing
		case "json":
			log.SetFormatter(&log.JSONFormatter{})
		default:
			return fmt.Errorf("unsupported log format '%s'", logFormat)
		}

		if logFile, _ := cmd.Flags().GetString("log"); logFile != "" {
			out, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}

			log.SetOutput(out)
		}

		rootDir, _ := cmd.Flags().GetString("root")
		if _, err := os.Stat(rootDir); errors.Is(err, fs.ErrNotExist) {
			if err := os.MkdirAll(rootDir, 0o700); err != nil {
				return err
			}

			xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
			if xdgRuntimeDir != "" && strings.HasPrefix(rootDir, xdgRuntimeDir) {
				// $XDG_RUNTIME_DIR defines the base directory relative to which user-specific non-essential runtime files and other file objects (such as sockets, named pipes, ...) should be stored. The directory MUST be owned by the user, and he MUST be the only one having read and write access to it. Its Unix access mode MUST be 0700. [...] Files in this directory MAY be subjected to periodic clean-up. To ensure that your files are not removed, they should have their access time timestamp modified at least once every 6 hours of monotonic time or the 'sticky' bit should be set on the file.
				err := os.Chmod(rootDir, os.FileMode(0o700)|os.ModeSticky)
				if err != nil {
					return err
				}
			}
		}

		if debug, _ := cmd.Flags().GetBool("debug"); debug {
			out, err := os.OpenFile(filepath.Join(rootDir, "debug.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}

			log.SetLevel(log.DebugLevel)

			log.AddHook(&writer.Hook{
				Writer:    out,
				LogLevels: log.AllLevels,
			})

			log.WithFields(log.Fields{
				"rawArgs": os.Args,
			}).Debug("invoking command")
		}

		return nil
	},
}

func init() {
	rootDir := filepath.Join("/run", programName)
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")

	if xdgRuntimeDir != "" && os.Getuid() != 0 {
		rootDir = filepath.Join(xdgRuntimeDir, programName)
	}

	rootCmd.PersistentFlags().String("root", rootDir, "root directory")
	rootCmd.PersistentFlags().String("log", "", "log file (default \"/dev/stderr\")")
	rootCmd.PersistentFlags().String("log-format", "text", "log format")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug logging")
}

// logToStderr returns true when the logger is configured to write to stderr, false otherwise.
func logToStderr() bool {
	l, ok := log.StandardLogger().Out.(*os.File)
	return ok && l.Fd() == os.Stderr.Fd()
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)

		if !logToStderr() {
			fmt.Fprintln(os.Stderr, err)
		}

		os.Exit(1)
	}
}
