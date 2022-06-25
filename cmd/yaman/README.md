# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][] that does not need elevated privileges. It was written for educational purposes.

When Yaman is executed by an unprivileged user, [fuse-overlayfs][] is used to mount the root filesystem (`rootfs`). As for networking, [slirp4netns][] is used for both unprivileged and privileged executions (no reason to use slirp4netns for "rootful" containers except simplicity).

Yaman supports the following registries:

- [Docker Hub](https://hub.docker.com/)
- [Red Hat Quay](https://quay.io/)

## Commands

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

⚠️ You must have a recent version of [fuse-overlayfs][] installed. When `fuse-overlayfs` is not installed, Yaman will fallback to native OverlayFS but that usually requires elevated privileges (with `sudo` for example).

### `yaman container`

Manage containers.

Alias: `yaman c`

#### `yaman container run`

**Note:** Yaman requires the use of fully qualified image names.

Let's run the image named [`docker.io/willdurand/hello-world`][hello-world]. This is a simple example inspired by Docker's [hello-world][hello-world-docker].

```console
$ yaman c run docker.io/willdurand/hello-world

Hello from @willdurand!

This message shows that your installation appears to be working correctly
(but that might be a lie because this is bleeding edge technology).

To generate this message, Yaman took the following steps:
 1. Yaman pulled the "willdurand/hello-world" image from the Docker Hub.
 2. Yaman created a new container from that image which runs the executable
    that produces the output you are currently reading. Under the hood,
    a "shim" named Yacs has been executed. This is the tool responsible
    for monitoring the container (which was created by a third tool: Yacr,
    an "OCI runtime").
 3. Yaman connected to the container output (via the shim), which sent it
    to your terminal. Amazing, right?

To try something more ambitious, you can run an Alpine container with:
 $ yaman c run -it docker.io/library/alpine sh

That's basically it because this is a learning project :D

For more examples and ideas, visit:
 https://github.com/willdurand/containers

```

##### `--detach` | `-d`

Run a container in the background with `--detach`:

```console
$ yaman c run -d docker.io/library/alpine:latest sleep 1000
2be09afa2b3b47c2a9975017aa2913fc
```

##### `--hostname`

By default, Yaman sets the container ID as hostname. You can configure a different value with `--hostname`:

```console
$ yaman c run --hostname="hello" docker.io/library/alpine:latest -- hostname
hello
```

##### `--interactive` | `-i`

Create an interactive container (that keeps `stdin` open) with `--interactive`:

```console
$ echo 'hello there' | yaman c run --interactive docker.io/library/alpine -- cat
hello there
```

##### `--tty` | `-t`

Spawn an interactive shell with both `-i` and `--tty` (short version: `-t`):

```console
$ yaman c run -it docker.io/library/alpine sh
/ # id
uid=0(root) gid=0(root) groups=0(root)
/ # hostname
56823f2c913b4d96a0b1b4ba6d978734
/ # ^C
/ # exit
```

##### Other options

| Option      | Description                                       |
| ----------- | ------------------------------------------------- |
| `--name`    | Assign a name to the container                    |
| `--rm`      | Automatically remove the container when it exits  |
| `--runtime` | Specify the OCI runtime to use for this container |

##### Example with Red Hat Quay

```console
$ yaman c run -it quay.io/aptible/alpine
/ # cat /etc/alpine-release
3.3.3
/ # exit

$ yaman c list -a
CONTAINER ID                       IMAGE                           COMMAND   STATUS                    NAME
b2985e49d1f34d539599bba4fc0e789d   quay.io/aptible/alpine:latest   /bin/sh   Exited (0) 1 second ago   sad_mclean
```

#### `yaman container inspect`

```console
$ yaman c inspect 2be09afa2b3b47c2a9975017aa2913fc
```

<details>
<summary>click to reveal the JSON output</summary>

```json
{
  "Id": "86f57361baf946f7b5e3d20b5fdde4ae",
  "Root": "/run/user/1000/yaman/containers/86f57361baf946f7b5e3d20b5fdde4ae",
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
      "path": "/run/user/1000/yaman/containers/86f57361baf946f7b5e3d20b5fdde4ae/rootfs"
    },
    "hostname": "86f57361baf946f7b5e3d20b5fdde4ae",
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
    "hooks": {
      "createRuntime": [
        {
          "path": "/workspace/containers/bin/yaman",
          "args": [
            "/workspace/containers/bin/yaman",
            "container",
            "hook",
            "network-setup"
          ]
        }
      ]
    },
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
          "type": "ipc"
        },
        {
          "type": "mount"
        },
        {
          "type": "network"
        },
        {
          "type": "pid"
        },
        {
          "type": "user"
        },
        {
          "type": "uts"
        }
      ]
    }
  },
  "Options": {
    "Name": "upbeat_mclaren",
    "Command": [
      "sleep",
      "1000"
    ],
    "Remove": true,
    "Hostname": "",
    "Interactive": false,
    "Tty": false,
    "Detach": true
  },
  "Created": "2022-06-25T14:16:12.982091802Z",
  "Started": "2022-06-25T14:16:13.020406768Z",
  "Exited": "0001-01-01T00:00:00Z",
  "Image": {
    "Hostname": "docker.io",
    "Name": "library/alpine",
    "Version": "latest",
    "BaseDir": "/run/user/1000/yaman/images/docker.io/library/alpine/latest",
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
    "ID": "86f57361baf946f7b5e3d20b5fdde4ae",
    "Runtime": "yacr",
    "State": {
      "ociVersion": "1.0.2",
      "id": "86f57361baf946f7b5e3d20b5fdde4ae",
      "status": "running",
      "pid": 103614,
      "bundle": "/run/user/1000/yaman/containers/86f57361baf946f7b5e3d20b5fdde4ae"
    },
    "Status": {},
    "Options": {
      "Runtime": "yacr"
    },
    "SocketPath": "/run/user/1000/yacs/86f57361baf946f7b5e3d20b5fdde4ae/shim.sock"
  }
}
```

</details>

#### `yaman container stop`

```console
$ yaman c stop 2be09afa2b3b47c2a9975017aa2913fc
```

#### `yaman container list`

List all containers and not only those currently running:

```console
$ yaman c list --all
CONTAINER ID                       IMAGE                             COMMAND    STATUS                          NAME
1234e4a90ed042dc96b1a6f80417b75a   docker.io/library/alpine:latest   hostname   Exited (0) About a minute ago   great_hermann
```

#### `yaman container delete`

```console
$ yaman c delete 2be09afa2b3b47c2a9975017aa2913fc
```

#### `yaman container attach`

```console
$ yaman c run -d --rm docker.io/library/alpine -- top -b
4bd06a2046e44e1d96c636c7ecae62d4

$ yaman c attach 4bd06a2046e44e1d96c636c7ecae62d4
Mem: 719244K used, 280824K free, 12244K shrd, 28092K buff, 407436K cached
CPU:   0% usr   0% sys   0% nic  95% idle   0% io   0% irq   4% sirq
Load average: 0.17 0.07 0.07 2/177 5
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1596   0%   0   0% top -b
Mem: 719204K used, 280864K free, 12268K shrd, 28100K buff, 407620K cached
CPU:   0% usr   0% sys   0% nic  99% idle   0% io   0% irq   0% sirq
Load average: 0.16 0.07 0.07 1/177 5
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1596   0%   0   0% top -b
Mem: 719724K used, 280344K free, 12268K shrd, 28108K buff, 407620K cached
CPU:   0% usr   0% sys   0% nic  99% idle   0% io   0% irq   0% sirq
Load average: 0.15 0.07 0.07 2/183 5
  PID  PPID USER     STAT   VSZ %VSZ CPU %CPU COMMAND
    1     0 root     R     1596   0%   0   0% top -b
^C
```

**Note:** the container is not stopped when we leave the attached container. This is a known limitation due to the fact that Yaman does not proxy the signals to the container process.

We can also attach a container that was created with a terminal (PTY):

```console
$ yaman c run -it -d --rm docker.io/library/alpine -- sh
a932b1afa47341d183abf16d36aa33dd

$ yaman c attach a932b1afa47341d183abf16d36aa33dd
/ #
```

Exiting this "attach session" will terminate the container process.

**Note:** there is currently no way to detach from the "attach session" and keep the container running. This would require a special key sequence (like Docker's `--detach-keys`).

### `yaman image`

Manage OCI images.

Alias: `yaman i`

#### `yaman image pull`

```console
$ yaman i pull docker.io/willdurand/hello-world
sha256:1bc1a702b0483184d0c0e12a9b3bfc20f3a89ed49b52fd8ad9d32c8180f01443
```

#### `yaman image list`

```console
$ yaman image list
NAME                     TAG         CREATED                PULLED           REGISTRY
willdurand/hello-world   latest      2022-06-17T20:32:59Z   3 seconds ago    docker.io
aptible/alpine           latest      2022-06-14T17:29:19Z   58 seconds ago   quay.io
library/alpine           latest      2022-05-23T19:19:31Z   2 minutes ago    docker.io
```

## Example with `runc`

Yaman uses [Yacs](../yacs/README.md) to "monitor" containers under the hood, which is capable of interacting with any OCI-compliant runtime. The default runtime is [Yacr](../yacr/README.md) but we can use the `--runtime` option to specify a different OCI runtime.

This is an example with `runc`:

```console
$ yaman c run --rm -it --runtime=runc docker.io/library/alpine sh
/ # hostname
af5479f3265a49f78569b8b65c7d1412
```

In a different terminal, we can query `runc` manually and see the container above listed:

```console
$ runc list
ID                                 PID         STATUS      BUNDLE                                                             CREATED                          OWNER
f6834e0a03134e50907f412fa6e394aa   2367        running     /run/user/1000/yaman/containers/f6834e0a03134e50907f412fa6e394aa   2022-06-20T06:48:58.759394476Z   vagrant
```

`runc` being the reference implementation and a production-ready tool, it has more features. For example, we can `exec` an existing container:

```console
$ runc exec -t af5479f3265a49f78569b8b65c7d1412 sh
/ # echo "hello" > /some-file
/ #
```

If we go back to the terminal where `yaman` is running, we should be able to see the newly created file and output its content:

```console
$ yaman c run --rm -it --runtime=runc docker.io/library/alpine sh
/ # hostname
af5479f3265a49f78569b8b65c7d1412
/ # ls
bin        home       mnt        root       some-file  tmp
dev        lib        opt        run        srv        usr
etc        media      proc       sbin       sys        var
/ # cat some-file
hello
/ #
```

[docker]: https://docs.docker.com/reference/
[fuse-overlayfs]: https://github.com/containers/fuse-overlayfs
[hello-world-docker]: https://hub.docker.com/_/hello-world
[hello-world]: https://hub.docker.com/r/willdurand/hello-world
[podman]: https://docs.podman.io/en/latest/
[slirp4netns]: https://github.com/rootless-containers/slirp4netns
