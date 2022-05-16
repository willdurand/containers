package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/willdurand/containers/yacr/containers"
)

func init() {
	cmd := &cobra.Command{
		Use:          "delete <id>",
		Short:        "Delete a container",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			force, _ := cmd.Flags().GetBool("force")

			rootDir, _ := cmd.Flags().GetString("root")
			container, err := containers.Load(rootDir, id)
			if err != nil {
				if force {
					log.WithFields(log.Fields{
						"id": id,
					}).Debug("delete: force deleted container")
					os.RemoveAll(filepath.Join(rootDir, id))
					return nil
				}

				return fmt.Errorf("delete: %w", err)
			}

			if !force && !container.IsStopped() {
				return fmt.Errorf("delete: unexpected status '%s' for container '%s'", container.State().Status, container.ID())
			}

			// Attempt to unmount all mountpoints recursively.
			err = syscall.Unmount(container.Rootfs(), syscall.MNT_DETACH)
			if err == nil {
				log.WithField("id", container.ID()).Debug("delete: unmounted rootfs")

				// On Gitpod with containerd alone and `--snapshotter=native`,
				// it seems to use shiftfs and there is a problem with empty
				// directories not removed that causes containerd to not delete
				// the rootfs directory. That's a problem because we cannot
				// the same `ctr run` twice or more...
				// Let's try to delete the directories if they still exist.
				for i := len(container.Spec().Mounts) - 1; i >= 0; i-- {
					mountpoint := container.Rootfs() + container.Spec().Mounts[i].Destination
					if err := os.Remove(mountpoint); err != nil {
						log.WithFields(logrus.Fields{
							"id":         container.ID(),
							"mountpoint": mountpoint,
							"error":      err,
						}).Debug("delete: rmdir()")
					}
				}
			} else {
				for _, dev := range []string{
					"/dev/null",
					"/dev/zero",
					"/dev/full",
					"/dev/random",
					"/dev/urandom",
					"/dev/tty",
				} {
					mountpoint := filepath.Join(container.Spec().Root.Path, dev)
					if err := syscall.Unmount(mountpoint, 0); err != nil {
						log.WithFields(logrus.Fields{
							"id":         container.ID(),
							"mountpoint": mountpoint,
							"error":      err,
						}).Warn("delete: unmount() failed")
					}
				}

				for i := len(container.Spec().Mounts) - 1; i >= 0; i-- {
					mountpoint := container.Rootfs() + container.Spec().Mounts[i].Destination
					if err := syscall.Unmount(mountpoint, syscall.MNT_DETACH); err != nil {
						log.WithFields(logrus.Fields{
							"id":         container.ID(),
							"mountpoint": mountpoint,
							"error":      err,
						}).Warn("delete: unmount() failed")
					}
				}
			}

			if err := container.Destroy(); err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#poststop
			if err := container.ExecuteHooks("Poststop"); !force && err != nil {
				return fmt.Errorf("delete: %w", err)
			}

			log.WithFields(log.Fields{
				"id": container.ID(),
			}).Info("delete: ok")

			return nil
		},
		Args: cobra.ExactArgs(1),
	}
	cmd.Flags().BoolP("force", "f", false, "force delete a container")

	rootCmd.AddCommand(cmd)
}
