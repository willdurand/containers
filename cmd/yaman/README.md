# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][] that does not need `root`-like privileges.

Yaman supports the following registries:

- [Docker Hub](https://hub.docker.com/)
- [Red Hat Quay](https://quay.io/)

## Commands

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

⚠️ You must have a recent version of [fuse-overlay][] installed. When `fuse-overlay` is not installed, Yaman will fallback to native OverlayFS but that usually requires elevated privileges (with `sudo` for example).

### `yaman container`

Manage containers.

Alias: `yaman c`

#### `yaman container run`

**Note:** Yaman requires the use of fully qualified image names.

Let's run the image named [`docker.io/willdurand/hello-world`][hello-world]. This is a simple example inspired by Docker's [hello-world][hello-world-docker].

``` console
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

Run a container in the background with `--detach` (short version: `-d`):

``` console
$ yaman c run -d docker.io/library/alpine:latest sleep 1000
2be09afa2b3b47c2a9975017aa2913fc
```

Change the container's hostname with `--hostname`:

``` console
$ yaman c run --hostname="hello" docker.io/library/alpine:latest -- hostname
hello
```

Create an interactive container (that keeps `stdin` open) with `--interactive` (short version: `-i`):

``` console
$ echo 'hello there' | yaman c run --interactive docker.io/library/alpine -- cat
hello there
^C

$ yaman c list
CONTAINER ID                       IMAGE                             COMMAND   STATUS      NAME
103493075a744ec0b47a3a9a6aed473e   docker.io/library/alpine:latest   cat       running     elated_bell
```

Spawn an interactive shell with both `-i` and `--tty` (short version: `-t`):

``` console
$ yaman c run -it docker.io/library/alpine sh
/ # id
uid=0(root) gid=0(root) groups=0(root)
/ # hostname
56823f2c913b4d96a0b1b4ba6d978734
/ # ^C
/ # exit
```

Use a different registry:

``` console
$ yaman c run -it quay.io/aptible/alpine
/ # cat /etc/alpine-release
3.3.3
/ # exit

$ yaman c list -a
CONTAINER ID                       IMAGE                           COMMAND   STATUS                    NAME
b2985e49d1f34d539599bba4fc0e789d   quay.io/aptible/alpine:latest   /bin/sh   Exited (0) 1 second ago   sad_mclean
```

#### `yaman container inspect`

``` console
$ yaman c inspect 2be09afa2b3b47c2a9975017aa2913fc
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

``` console
$ yaman c stop 2be09afa2b3b47c2a9975017aa2913fc
```

#### `yaman container list`

List all containers and not only those currently running:

``` console
$ yaman c list --all
CONTAINER ID                       IMAGE                             COMMAND    STATUS                          NAME
1234e4a90ed042dc96b1a6f80417b75a   docker.io/library/alpine:latest   hostname   Exited (0) About a minute ago   great_hermann
```

#### `yaman container delete`

``` console
$ yaman c delete 2be09afa2b3b47c2a9975017aa2913fc
```

#### `yaman container attach`

``` console
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

``` console
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

``` console
$ yaman image pull docker.io/library/hello-world
downloaded docker.io/library/hello-world:latest
```

#### `yaman image list`

``` console
$ yaman image list
NAME                  TAG         CREATED                PULLED           REGISTRY
library/hello-world   latest      2021-09-23T23:47:57Z   12 minutes ago   docker.io
library/redis         latest      2022-06-13T20:08:18Z   10 minutes ago   docker.io
```

## Example with `runc`

Yaman uses the [yacs](../yacs/README.md) shim under the hood, which should be able to interact with an OCI-compliant runtime even though the default runtime is [yacr](../yacr/README.md). We can pass the `--runtime` option to `yaman container run` in order to use a different OCI runtime.

This is an example with `runc`:

``` console
$ yaman c run --rm -it --runtime=runc docker.io/library/alpine sh
/ # hostname
af5479f3265a49f78569b8b65c7d1412
```

In a different terminal, we can query `runc` manually and see the container above listed:

``` console
$ runc list
ID                                 PID         STATUS    BUNDLE                                                   CREATED                          OWNER
af5479f3265a49f78569b8b65c7d1412   3394        running   /tmp/yaman/containers/af5479f3265a49f78569b8b65c7d1412   2022-06-17T20:06:38.326130089Z   root
```

`runc` being the reference implementation and a production-ready tool, it has more features. For example, we can `exec` an existing container:

``` console
$ runc exec -t af5479f3265a49f78569b8b65c7d1412 sh
/ # echo "hello" > /some-file
/ #
```

If we go back to the terminal where `yaman` is running, we should be able to see the newly created file and output its content:

``` console
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
[hello-world-docker]: https://hub.docker.com/_/hello-world
[hello-world]: https://hub.docker.com/r/willdurand/hello-world
[podman]: https://docs.podman.io/en/latest/
