# Yet Another Container Runtime\*

\* = unsafe, do not use in production!

## Important

This runtime is known to be unsafe because:

1. I wrote it to learn about container runtimes
2. it is very likely vulnerable to [CVE-2019-5736][] and probably other issues fixed in [runc][] already
3. it is imcomplete, not unit tested and unreviewed

## Getting started (standalone)

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

First, we need an OCI bundle, which we can create using `docker` and `runc`:

```
$ make alpine_bundle
```

> Note: we pass `--rootless` to `runc spec` to generate a configuration for "rootless containers". Under the hood, this creates some [user namespace mappings][].

Edit the `config.json` file to set `process.terminal` to `false` and update the `process.args` to execute `/bin/sleep 1000`:

```diff
--- a/config.json
+++ b/config.json

 {
        "ociVersion": "1.0.2-dev",
        "process": {
-               "terminal": true,
+               "terminal": false,
                "user": {
                        "uid": 0,
                        "gid": 0
                },
                "args": [
-                       "sh"
+                       "/bin/sleep", "1000"
                ],
                "env": [
                        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
```

Now we can use `yacr create` to create a new container. We need to pass a container ID and the path to the bundle we made previously. In order to create a "rootless container", we need to specify a root directory (i.e. the location used by `yacr` to write its data) that is accessible by the current user. If you don't have the `XDG_RUNTIME_DIR` environment variable configured on your system (see: [XDG Base Directory][]), you'll have to specify the root directory (e.g. `--root /tmp/yacr`).

```
$ yacr create test-id --bundle .
ERRO[0000] container: failed to mount filesystem         destination=/tmp/alpine-bundle/rootfs/sys/fs/cgroup error="operation not permitted" id=test-id options="[nosuid noexec nodev relatime ro]" source=cgroup type=cgroup
INFO[0000] create: ok                                    id=test-id
```

> Note: you can ignore the error about `cgroup` because `yacr` doesn't support cgroups (yet).

Creating a container should not execute its process right away. Instead, it should spawn a new containerized process and wait for the "start" command. We can check the containers managed with `yacr` by running `yacr list`:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
test-id     created     2022-05-30T22:00:00Z   137261      /tmp/alpine-bundle
```

We can now start the container with `yacr start`:

```
$ yacr start test-id
INFO[0022] container: executing process                  id=test-id processArgs="[/bin/sleep 1000]"
INFO[0000] start: (probably) ok                          id=test-id
```

The container should now be running, which we can confirm with `yacr list` again:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
test-id     running     2022-05-30T22:00:00Z   137261      /tmp/alpine-bundle
```

We should also see our process running on the host machine with `ps`:

```
$ ps auxf
USER      PID     %CPU %MEM  VSZ    RSS TTY      STAT START   TIME COMMAND
[...]
gitpod    137261  0.0  0.0   1596     4 pts/2    S    22:01   0:00 /bin/sleep 1000
```

The PID reported by `yacr list` matches the `ps` output above. The owner of the process is `gitpod` because we execute a "rootless container" 😎. Note that if you execute `yacr` with `sudo`, the owner would be `root`.

Since `yacr` implements the [runtime-spec][], we can send a signal to the process with `yacr kill`:

```
$ yacr kill test-id 9
INFO[0000] kill: ok                                      id=test-id signal=9
```

Let's check:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
test-id     stopped     2022-05-30T22:00:00Z   0           /tmp/alpine-bundle
```

The container has been stopped because we sent the `SIGKILL` signal. It shouldn't appear in the `ps` output anymore.

It is now safe to delete the container with `yacr delete`:

```
$ yacr delete test-id
INFO[0000] delete: ok                                    id=test-id
```

At this point, `yacr list` should not list the container with ID `test-id` anymore either.

### Spawning a shell

In the previous section, the `config.json` has been edited to disable the `terminal` support and execute `sleep` instead of `sh` in the container. Let's revert those changes to create a new container that will spawn a shell.

First, we need a receiver terminal. We will use [`recvtty`][recvtty] from the [runc][] repository:

```
$ go install github.com/opencontainers/runc/contrib/cmd/recvtty@latest
$ recvtty /tmp/console.sock
```

In another terminal, let's get back to the `/tmp/alpine-bundle` directory and create the container with the `--console-socket` option:

```
$ yacr create test-id --bundle . --console-socket /tmp/console.sock
INFO[0000] create: ok                                    id=test-id
```

When we start the container, the runtime will return immediately:

```
$ yacr start test-id
INFO[0000] start: (probably) ok                          id=test-id
```

In the terminal executing `recvtty`, we should now see `sh` running in the container:

```
/ # ps
ps
PID   USER     TIME  COMMAND
    1 root      0:00 sh
   19 root      0:00 ps
```

## Getting started with Docker

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

Let's create a new Docker daemon with the `yacr` runtime:

```
$ ./scripts/run-dockerd
```

In another terminal, you can connect to this daemon by running `docker` with `-H unix:///tmp/d2/d2.socket` or use the `./scripts/docker` wrapper in this repository:

```
$ ./scripts/docker info
[...]
 Runtimes: gitpod io.containerd.runc.v2 io.containerd.runtime.v1.linux runc yacr
 Default Runtime: yacr
[...]
```

We can then use Docker as usual:

```
$ ./scripts/docker run --rm -it busybox:latest /bin/sh
Unable to find image 'busybox:latest' locally
latest: Pulling from library/busybox
cecc78ee4075: Pull complete 
Digest: sha256:de56395ae0788e364797f0c60464d4693c43c33cc04ec26fc3b0931b2e7c9d7d
Status: Downloaded newer image for busybox:latest
/ # ps
PID   USER     TIME  COMMAND
    1 root      0:00 /bin/sh
   25 root      0:00 ps
/ # ping -c 1 1.1.1.1
PING 1.1.1.1 (1.1.1.1): 56 data bytes
64 bytes from 1.1.1.1: seq=0 ttl=59 time=6.093 ms

--- 1.1.1.1 ping statistics ---
1 packets transmitted, 1 packets received, 0% packet loss
round-trip min/avg/max = 6.093/6.093/6.093 ms
/ #
```

> Note: `docker exec` does not work currently.

## Getting started with containerd

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

First, [install `containerd`][install-containerd], then run `containerd` with elevated privileges:

```
$ sudo containerd
```

In order to run containers with the `yacr` runtime, we need an image first:

```
$ sudo ctr images pull docker.io/library/alpine:latest
```

We can now run a new container with our runtime by specifying its path with `--runc-binary` (we'll use the default "runc shim"). On Gitpod, we should also pass `--snapshotter=native` to `ctr run`. The command below will spawn an interactive shell in our container thanks to the `--tty` option:

```
$ sudo ctr run --runc-binary=/workspace/containers/bin/yacr --snapshotter=native --tty docker.io/library/alpine:latest alpine-1 /bin/sh
/ #
```

In a different terminal, we can see list the containers with `yacr list`:

```
$ sudo ./bin/yacr --root /run/containerd/runc/default list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    running     2022-05-30T22:00:00Z   18166       /run/containerd/io.containerd.runtime.v2.task/default/alpine-1
```

[cve-2019-5736]: https://unit42.paloaltonetworks.com/breaking-docker-via-runc-explaining-cve-2019-5736/
[install-containerd]: https://github.com/containerd/containerd/blob/main/docs/getting-started.md
[recvtty]: https://github.com/opencontainers/runc/blob/main/contrib/cmd/recvtty/recvtty.go
[runc]: https://github.com/opencontainers/runc/
[runtime-spec]: https://github.com/opencontainers/runtime-spec
[user namespace mappings]: https://github.com/opencontainers/runtime-spec/blob/27924127bf391ea7691924c6dcb01f3369d69fe2/config-linux.md#user-namespace-mappings
[xdg base directory]: https://specifications.freedesktop.org/basedir-spec/basedir-spec-latest.html
