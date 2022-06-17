# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][].

## Commands

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

⚠️ You must have "root" privileges to use `yaman` because the tool needs to mount the "rootfs" of a container as an Overlay FS. You can use `sudo yaman` as shown in the next sections.

### `yaman image`

Manage OCI images.

#### `yaman image pull`

```
$ sudo yaman image pull docker.io/library/hello-world
downloaded docker.io/library/hello-world:latest
```

#### `yaman image list`

```
$ sudo yaman image list
NAME                  TAG         CREATED                PULLED           REGISTRY
library/hello-world   latest      2021-09-23T23:47:57Z   12 minutes ago   docker.io
library/redis         latest      2022-06-13T20:08:18Z   10 minutes ago   docker.io
```

### `yaman container`

Alias: `yaman c`

#### `yaman container run`

**Note:** Yaman uses fully qualified image names although it currently only supports images listed on the Docker registry.

This is a simple example:

```
$ sudo yaman c run docker.io/library/hello-world

Hello from Docker!
This message shows that your installation appears to be working correctly.

To generate this message, Docker took the following steps:
 1. The Docker client contacted the Docker daemon.
 2. The Docker daemon pulled the "hello-world" image from the Docker Hub.
    (amd64)
 3. The Docker daemon created a new container from that image which runs the
    executable that produces the output you are currently reading.
 4. The Docker daemon streamed that output to the Docker client, which sent it
    to your terminal.

To try something more ambitious, you can run an Ubuntu container with:
 $ docker run -it ubuntu bash

Share images, automate workflows, and more with a free Docker ID:
 https://hub.docker.com/

For more examples and ideas, visit:
 https://docs.docker.com/get-started/

```

Run a container in the background with `--detach` (short version: `-d`):

```
$ sudo yaman c run -d docker.io/library/alpine:latest sleep 1000
2be09afa2b3b47c2a9975017aa2913fc
```

Change the container's hostname with `--hostname`:

```
$ sudo yaman c run --hostname="hello" docker.io/library/alpine:latest -- hostname
hello
```

Create an interactive container (that keeps `stdin` open) with `--interactive` (short version: `-i`):

```
$ echo 'hello there' | sudo yaman c run --interactive docker.io/library/alpine -- cat
hello there
^C

$ sudo yaman c list
CONTAINER ID                       IMAGE                             COMMAND   STATUS      NAME
103493075a744ec0b47a3a9a6aed473e   docker.io/library/alpine:latest   cat       running     elated_bell
```

Spawn an interactive shell with both `-i` and `--tty` (short version: `-t`):

```
$ sudo yaman c run -it docker.io/library/alpine sh
/ # id
uid=0(root) gid=0(root) groups=0(root)
/ # hostname
56823f2c913b4d96a0b1b4ba6d978734
/ # ^C
/ # exit
```

#### `yaman container inspect`

```
$ sudo yaman c inspect 2be09afa2b3b47c2a9975017aa2913fc
```

<details>
<summary>click to reveal the JSON output</summary>

```json
{
  "Id": "2be09afa2b3b47c2a9975017aa2913fc",
  "Root": "/run/yaman/containers/2be09afa2b3b47c2a9975017aa2913fc",
  "Config": {
    "ociVersion": "1.0.2",
    "process": {
      "user": {
        "uid": 0,
        "gid": 0
      },
      "args": [
        "sleep",
        "1000"
      ],
      "env": [
        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
      ],
      "cwd": "/"
    },
    "root": {
      "path": "/run/yaman/containers/2be09afa2b3b47c2a9975017aa2913fc/rootfs"
    },
    "hostname": "2be09afa2b3b47c2a9975017aa2913fc",
    "mounts": [
      {
        "destination": "/proc",
        "type": "proc",
        "source": "proc"
      },
      {
        "destination": "/dev",
        "type": "tmpfs",
        "source": "tmpfs",
        "options": [
          "nosuid",
          "strictatime",
          "mode=755",
          "size=65536k"
        ]
      },
      {
        "destination": "/dev/pts",
        "type": "devpts",
        "source": "devpts",
        "options": [
          "nosuid",
          "noexec",
          "newinstance",
          "ptmxmode=0666",
          "mode=0620"
        ]
      },
      {
        "destination": "/dev/shm",
        "type": "tmpfs",
        "source": "shm",
        "options": [
          "nosuid",
          "noexec",
          "nodev",
          "mode=1777",
          "size=65536k"
        ]
      },
      {
        "destination": "/dev/mqueue",
        "type": "mqueue",
        "source": "mqueue",
        "options": [
          "nosuid",
          "noexec",
          "nodev"
        ]
      },
      {
        "destination": "/sys",
        "type": "none",
        "source": "/sys",
        "options": [
          "rbind",
          "nosuid",
          "noexec",
          "nodev",
          "ro"
        ]
      }
    ],
    "linux": {
      "uidMappings": [
        {
          "containerID": 0,
          "hostID": 0,
          "size": 1
        }
      ],
      "gidMappings": [
        {
          "containerID": 0,
          "hostID": 0,
          "size": 1
        }
      ],
      "namespaces": [
        {
          "type": "pid"
        },
        {
          "type": "ipc"
        },
        {
          "type": "uts"
        },
        {
          "type": "mount"
        },
        {
          "type": "user"
        }
      ]
    }
  },
  "Options": {
    "Name": "jovial_banach",
    "Command": [
      "sleep",
      "1000"
    ],
    "Remove": false,
    "Hostname": "",
    "Tty": false
  },
  "Created": "2022-06-16T22:44:20.94104793+02:00",
  "Started": "2022-06-16T22:44:21.009890159+02:00",
  "Exited": "0001-01-01T00:00:00Z",
  "Image": {
    "Hostname": "docker.io",
    "Name": "library/alpine",
    "Version": "latest",
    "BaseDir": "/run/yaman/images/docker.io/library/alpine/latest",
    "Config": {
      "created": "2022-05-23T19:19:31.970967174Z",
      "architecture": "amd64",
      "os": "linux",
      "config": {
        "Env": [
          "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
        ],
        "Cmd": [
          "/bin/sh"
        ]
      },
      "rootfs": {
        "type": "layers",
        "diff_ids": [
          "sha256:24302eb7d9085da80f016e7e4ae55417e412fb7e0a8021e95e3b60c67cde557d"
        ]
      },
      "history": [
        {
          "created": "2022-05-23T19:19:30.413290187Z",
          "created_by": "/bin/sh -c #(nop) ADD file:8e81116368669ed3dd361bc898d61bff249f524139a239fdaf3ec46869a39921 in / "
        },
        {
          "created": "2022-05-23T19:19:31.970967174Z",
          "created_by": "/bin/sh -c #(nop)  CMD [\"/bin/sh\"]",
          "empty_layer": true
        }
      ]
    },
    "Manifest": {
      "schemaVersion": 2,
      "mediaType": "application/vnd.docker.distribution.manifest.v2+json",
      "config": {
        "mediaType": "application/vnd.docker.container.image.v1+json",
        "digest": "sha256:e66264b98777e12192600bf9b4d663655c98a090072e1bab49e233d7531d1294",
        "size": 1472
      },
      "layers": [
        {
          "mediaType": "application/vnd.docker.image.rootfs.diff.tar.gzip",
          "digest": "sha256:2408cc74d12b6cd092bb8b516ba7d5e290f485d3eb9672efc00f0583730179e8",
          "size": 2798889
        }
      ]
    }
  },
  "Shim": {
    "State": {
      "ociVersion": "1.0.2",
      "id": "2be09afa2b3b47c2a9975017aa2913fc",
      "status": "running",
      "pid": 190288,
      "bundle": "/run/yaman/containers/2be09afa2b3b47c2a9975017aa2913fc"
    },
    "Status": {},
    "Options": {
      "Runtime": "yacr"
    },
    "SocketPath": "/run/yacs/2be09afa2b3b47c2a9975017aa2913fc/shim.sock"
  }
}
```
</details>

#### `yaman container stop`

```
$ sudo yaman c stop 2be09afa2b3b47c2a9975017aa2913fc
```

#### `yaman container list`

List all containers and not only those currently running:

```
$ sudo yaman c list --all
CONTAINER ID                       IMAGE                             COMMAND    STATUS                          NAME
1234e4a90ed042dc96b1a6f80417b75a   docker.io/library/alpine:latest   hostname   Exited (0) About a minute ago   great_hermann
```

#### `yaman container delete`

```
$ sudo yaman c delete 2be09afa2b3b47c2a9975017aa2913fc
```

[docker]: https://docs.docker.com/reference/
[podman]: https://docs.podman.io/en/latest/
