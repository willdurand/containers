package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
)

func LoadBundleConfig(bundle string) (runtimespec.Spec, error) {
	var spec runtimespec.Spec

	data, err := ioutil.ReadFile(filepath.Join(bundle, "config.json"))
	if err != nil {
		return spec, fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, fmt.Errorf("failed to parse config.json: %w", err)
	}

	return spec, nil
}

func BaseSpec(rootfs string) runtimespec.Spec {
	return runtimespec.Spec{
		Version: runtimespec.Version,
		Process: &runtimespec.Process{
			Terminal: false,
			User: runtimespec.User{
				UID: 0,
				GID: 0,
			},
			Args: []string{"sh"},
			Env:  []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Cwd:  "/",
		},
		Root: &runtimespec.Root{
			Path: rootfs,
		},
		Hostname: "container",
		Mounts: []runtimespec.Mount{
			{
				Destination: "/proc",
				Type:        "proc",
				Source:      "proc",
			},
			{
				Destination: "/dev",
				Type:        "tmpfs",
				Source:      "tmpfs",
				Options:     []string{"nosuid", "strictatime", "mode=755", "size=65536k"},
			},
			{
				Destination: "/dev/pts",
				Type:        "devpts",
				Source:      "devpts",
				Options:     []string{"nosuid", "noexec", "newinstance", "ptmxmode=0666", "mode=0620"},
			},
			{
				Destination: "/dev/shm",
				Type:        "tmpfs",
				Source:      "shm",
				Options:     []string{"nosuid", "noexec", "nodev", "mode=1777", "size=65536k"},
			},
			{
				Destination: "/dev/mqueue",
				Type:        "mqueue",
				Source:      "mqueue",
				Options:     []string{"nosuid", "noexec", "nodev"},
			},
			{
				Destination: "/sys",
				Type:        "none",
				Source:      "/sys",
				Options:     []string{"rbind", "nosuid", "noexec", "nodev", "ro"},
			},
		},
		Linux: &runtimespec.Linux{
			UIDMappings: []runtimespec.LinuxIDMapping{
				{ContainerID: 0, HostID: uint32(os.Getuid()), Size: 1},
			},
			GIDMappings: []runtimespec.LinuxIDMapping{
				{ContainerID: 0, HostID: uint32(os.Getgid()), Size: 1},
			},
			Namespaces: []runtimespec.LinuxNamespace{
				{Type: "pid"},
				{Type: "ipc"},
				{Type: "uts"},
				{Type: "mount"},
				{Type: "user"},
			},
		},
	}
}
