package runtime

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	runtimespec "github.com/opencontainers/runtime-spec/specs-go"
	"github.com/willdurand/containers/internal/user"
)

func LoadSpec(bundleDir string) (runtimespec.Spec, error) {
	var spec runtimespec.Spec

	data, err := ioutil.ReadFile(filepath.Join(bundleDir, "config.json"))
	if err != nil {
		return spec, fmt.Errorf("failed to read config.json: %w", err)
	}
	if err := json.Unmarshal(data, &spec); err != nil {
		return spec, fmt.Errorf("failed to parse config.json: %w", err)
	}

	return spec, nil
}

func BaseSpec(rootfs string, rootless bool) (*runtimespec.Spec, error) {
	mounts := []runtimespec.Mount{
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
	}

	resources := &runtimespec.LinuxResources{
		Devices: []runtimespec.LinuxDeviceCgroup{
			{
				Allow:  false,
				Access: "rwm",
			},
		},
	}

	namespaces := []runtimespec.LinuxNamespace{
		{Type: "ipc"},
		{Type: "mount"},
		{Type: "network"},
		{Type: "pid"},
		{Type: "uts"},
	}

	uidMappings := []runtimespec.LinuxIDMapping{}
	gidMappings := []runtimespec.LinuxIDMapping{}

	if rootless {
		mounts = append(mounts,
			runtimespec.Mount{
				Destination: "/sys",
				Type:        "none",
				Source:      "/sys",
				Options:     []string{"rbind", "nosuid", "noexec", "nodev", "ro"},
			},
		)

		// No resources in rootless mode.
		resources = nil

		namespaces = append(namespaces, runtimespec.LinuxNamespace{Type: "user"})

		uid, err := user.GetSubUid()
		if err != nil {
			return nil, err
		}

		uidMappings = append(
			uidMappings,
			runtimespec.LinuxIDMapping{
				ContainerID: 0,
				HostID:      uint32(os.Getuid()),
				Size:        1,
			},
			runtimespec.LinuxIDMapping{
				ContainerID: 1,
				HostID:      uint32(uid.ID),
				Size:        uint32(uid.Size),
			},
		)

		gid, err := user.GetSubGid()
		if err != nil {
			return nil, err
		}

		gidMappings = append(
			gidMappings,
			runtimespec.LinuxIDMapping{
				ContainerID: 0,
				HostID:      uint32(os.Getgid()),
				Size:        1,
			},
			runtimespec.LinuxIDMapping{
				ContainerID: 1,
				HostID:      uint32(gid.ID),
				Size:        uint32(gid.Size),
			},
		)
	}

	return &runtimespec.Spec{
		Version: runtimespec.Version,
		Process: &runtimespec.Process{
			Terminal: false,
			User: runtimespec.User{
				UID: 0,
				GID: 0,
			},
			Args: []string{"sleep", "100"},
			Env:  []string{"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"},
			Cwd:  "/",
			Capabilities: &runtimespec.LinuxCapabilities{
				Bounding: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
				},
				Effective: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
				},
				Inheritable: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
				},
				Permitted: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
				},
				Ambient: []string{
					"CAP_AUDIT_WRITE",
					"CAP_KILL",
					"CAP_NET_BIND_SERVICE",
				},
			},
			Rlimits: []runtimespec.POSIXRlimit{
				{
					Type: "RLIMIT_NOFILE",
					Hard: 1024,
					Soft: 1024,
				},
			},
			NoNewPrivileges: true,
		},
		Root: &runtimespec.Root{
			Path: rootfs,
		},
		Hostname: "container",
		Mounts:   mounts,
		Linux: &runtimespec.Linux{
			Resources:   resources,
			UIDMappings: uidMappings,
			GIDMappings: gidMappings,
			Namespaces:  namespaces,
			MaskedPaths: []string{
				"/proc/acpi",
				"/proc/asound",
				"/proc/kcore",
				"/proc/keys",
				"/proc/latency_stats",
				"/proc/timer_list",
				"/proc/timer_stats",
				"/proc/sched_debug",
				"/sys/firmware",
				"/proc/scsi",
			},
			ReadonlyPaths: []string{
				"/proc/bus",
				"/proc/fs",
				"/proc/irq",
				"/proc/sys",
				"/proc/sysrq-trigger",
			},
		},
	}, nil
}
