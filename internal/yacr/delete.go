package yacr

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/sirupsen/logrus"
	"github.com/willdurand/containers/internal/yacr/container"
)

func Delete(rootDir, containerId string, force bool) error {
	container, err := container.LoadWithBundleConfig(rootDir, containerId)
	if err != nil {
		if force {
			logrus.WithFields(logrus.Fields{
				"id": containerId,
			}).Debug("force deleted container")
			os.RemoveAll(filepath.Join(rootDir, containerId))
			return nil
		}

		return err
	}

	if !force && !container.IsStopped() {
		return fmt.Errorf("unexpected status '%s' for container '%s'", container.State.Status, container.ID())
	}

	// Attempt to unmount all mountpoints recursively.
	err = syscall.Unmount(container.Rootfs(), syscall.MNT_DETACH)
	if err == nil {
		logrus.WithField("id", container.ID()).Debug("unmounted rootfs")

		// On Gitpod with containerd alone and `--snapshotter=native`,
		// it seems to use shiftfs and there is a problem with empty
		// directories not removed that causes containerd to not delete
		// the rootfs directory. That's a problem because we cannot
		// the same `ctr run` twice or more...
		// Let's try to delete the directories if they still exist.
		for i := len(container.Spec.Mounts) - 1; i >= 0; i-- {
			mountpoint := container.Rootfs() + container.Spec.Mounts[i].Destination
			if err := os.Remove(mountpoint); err != nil {
				logrus.WithFields(logrus.Fields{
					"id":         container.ID(),
					"mountpoint": mountpoint,
					"error":      err,
				}).Debug("rmdir()")
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
			mountpoint := filepath.Join(container.Spec.Root.Path, dev)
			if err := syscall.Unmount(mountpoint, 0); err != nil {
				logrus.WithFields(logrus.Fields{
					"id":         container.ID(),
					"mountpoint": mountpoint,
					"error":      err,
				}).Warn("unmount() failed")
			}
		}

		for i := len(container.Spec.Mounts) - 1; i >= 0; i-- {
			mountpoint := container.Rootfs() + container.Spec.Mounts[i].Destination
			if err := syscall.Unmount(mountpoint, syscall.MNT_DETACH); err != nil {
				logrus.WithFields(logrus.Fields{
					"id":         container.ID(),
					"mountpoint": mountpoint,
					"error":      err,
				}).Warn("unmount() failed")
			}
		}
	}

	if err := container.Destroy(); err != nil {
		return err
	}

	// See: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config.md#poststop
	if err := container.ExecuteHooks("Poststop"); !force && err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"id": container.ID(),
	}).Info("ok")

	return nil
}
