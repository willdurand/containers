# Yet Another Container Shim

This is an example of a container shim that exposes an HTTP API[^1] to control the life cycle of a container process. Theoretically, shims should be a small as possible because container managers use a shim per container process. This isn't the case of this shim, though, but also no one should be using it except for learning purposes.

[^1]: this might change in the future (to a GRPC API with [ttrpc][])

## Getting started with an example

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

First, we need a new bundle:

```console
$ make alpine_bundle
```

Let's edit the `config.json` file generated in `/tmp/alpine-bundle` as follows:

```diff
--- a/config.json
+++ b/config.json
@@ -6,8 +6,7 @@
       "gid": 0
     },
     "args": [
-      "sleep",
-      "100"
+      "sh", "/hello-loop.sh"
     ],
     "env": [
       "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
```

We should also create this new file named `hello-loop.sh`:

```console
$ cat <<'EOF' > /tmp/alpine-bundle/rootfs/hello-loop.sh
#!/bin/sh

signal_handler() {
    >&2 echo "bye, bye"
    exit 123
}

trap 'signal_handler' TERM

while [ true ]; do
    echo "[$(date)] Hello!"
    sleep 1
done
EOF
```

This script loops forever and prints a message every second, unless it receives a `SIGTERM` signal, in which case it exits with a status equal to `123`.

We are ready to manually execute `yacs` (the shim) to create a new container with the `yacr` runtime (the default).

```console
$ yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
/home/gitpod/.run/yacs/alpine-1/shim.sock
```

**Note:** This step is usually automated by a container manager such as [Yaman][].

This should start a new shim that will automatically create a container process. We can check with `yacr list` and `ps`:

```console
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    created     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle

$ ps auxf
USER         PID    %CPU %MEM    VSZ     RSS TTY       STAT START   TIME COMMAND
[...]
gitpod       44458  0.0  0.0     1079856 7260 ?        Ssl  22:01   0:00 yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
gitpod       44488  0.0  0.0     1076520 5780 ?        Sl   22:01   0:00  \_ yacr --log-format json --log /home/gitpod/.run/yacs/alpine-1/yacr.log create container alpine-1 --root /home/gitpod/.run/yacr --bundle /tmp/alpine-bundle
```

When the command returns, it prints a unix socket address that can be used to query the shim using... HTTP. This isn't great but it is enough to demonstrate how a shim works.

We can use `curl` to interact with the shim:

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
{
  "ID": "alpine-1",
  "Runtime": "yacr",
  "State": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "created",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "Status": {}
}
```

We can now start the container by sending the `start` command (`cmd`) in a `POST` HTTP request:

```console
$ curl -X POST -d 'cmd=start' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
{
  "ID": "alpine-1",
  "Runtime": "yacr",
  "State": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "running",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "Status": {}
}
```

**Note:** [`jq`][jq] was used to pretty-print the JSON responses in the different examples.

The container is now running, which we can confirm with `yacr list` and `ps`:

```console
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    running     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle
```

```console
$ ps auxf
USER         PID    %CPU %MEM    VSZ     RSS TTY       STAT START   TIME COMMAND
[...]
gitpod       44458  0.0  0.0     1079856 7260 ?        Ssl  22:01   0:00 yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
gitpod       44488  0.0  0.0     1596    4    ?        S    22:01   0:00  \_ sh /hello-loop.sh
gitpod       55758  0.0  0.0     1596    4    ?        S    22:02   0:00      \_ sleep 1
```

We can query the shim to get the container's logs:

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/logs
{"m":"[Sun Jun 12 11:51:44 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:51:44.947554491Z"}
{"m":"[Sun Jun 12 11:51:45 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:51:45.948493454Z"}
{"m":"[Sun Jun 12 11:51:46 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:51:46.949371235Z"}
{"m":"[Sun Jun 12 11:51:47 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:51:47.950339068Z"}
```

Each entry is a JSON object with the following properties:

- `m`: the message
- `s`: the stream (either `stdout` or `stderr`)
- `t`: the timestamp

We can also use the shim HTTP API to send a signal to the container:

```console
$ curl -X POST -d 'cmd=kill' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
{
  "ID": "alpine-1",
  "Runtime": "yacr",
  "State": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "running",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "Status": {}
}
```

In the output above, we see the `status` set to `running` despite our attempt to "kill" the container. This happens because of timing, the container probably didn't have time to change its state.

If we query the logs again, we can see that the container actually received the signal (`SIGTERM` by default):

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/logs
[...]
{"m":"[Sun Jun 12 11:52:36 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:52:36.001548972Z"}
{"m":"[Sun Jun 12 11:52:37 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:52:37.002608715Z"}
{"m":"[Sun Jun 12 11:52:38 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:52:38.003658337Z"}
{"m":"bye, bye","s":"stderr","t":"2022-06-12T11:52:40.005199923Z"}
```

The container printed the message of the `signal_handler` defined in the `hello-loop.sh` script so the `kill` command worked as intended.

Let's query the state of the container again:

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
{
  "ID": "alpine-1",
  "Runtime": "yacr",
  "State": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "stopped",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "Status": {
    "exitStatus": 123,
    "exited": true,
    "pid": 44488,
    "waitStatus": 31488
  }
}
```

The container is now "stopped". The `exitStatus` is `123` and matches what we defined in the `hello-loop.sh` file created previously. Note also that the shim is still alive and we still have access to the container's full state and stdout/stderr logs. This is one of the reasons why shims are used.

```console
yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    stopped     2022-06-03T22:00:00Z   0           /tmp/alpine-bundle
```

We can now delete the container. This API request should not return anything (HTTP 204):

```console
$ curl -X POST -d 'cmd=delete' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
```

If we query the state of the shim again, it should indicate that the container does not exist anymore:

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
container does not exist
```

Finally, we can terminate the shim with a `DELETE` HTTP request:

```console
$ curl -X DELETE --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://shim/
BYE
```

## Getting started with `runc`

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

This shim should be able to use any OCI-compliant runtime like [`runc`][runc] (the reference implementation). Let's reproduce what was done in the previous section but with `runc`.

```console
$ yacs --bundle /tmp/alpine-bundle/ --container-id alpine-runc --runtime runc
/home/gitpod/.run/yacs/alpine-runc/shim.sock
```

The `ps` output below shows that `runc` has been invoked:

```console
$ ps auxf
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
[...]
gitpod      4321  0.0  0.0 1079876 6776 ?        Ssl  10:57   0:00 yacs --bundle /tmp/alpine-bundle/ --container-id alpine-runc --runtime runc
gitpod      4363  0.0  0.0 1083784 9980 ?        Ssl  10:57   0:00  \_ runc init
```

```console
$ curl -X POST -d 'cmd=start' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://shim/
{
  "id": "alpine-runc",
  "runtime": "runc",
  "state": {
    "ociVersion": "1.0.2-dev",
    "id": "alpine-runc",
    "status": "running",
    "pid": 4363,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {}
}
```

We could use `runc list` to list the containers created by `runc`:

```console
$ runc list
ID            PID         STATUS      BUNDLE               CREATED                          OWNER
alpine-runc   4363        running     /tmp/alpine-bundle   2022-06-05T10:57:18.334165274Z   gitpod
```

Since `runc` is the reference implementation and a production-ready runtime, it has a LOT more features than [`yacr`](../yacr/). For instance, we can use `runc exec` to execute a new process in the container, like spawning a shell:

```console
$ runc exec -t alpine-runc /bin/sh
/ # ps
PID   USER     TIME  COMMAND
    1 root      0:00 sh /hello-loop.sh
  229 root      0:00 /bin/sh
  236 root      0:00 sleep 1
  237 root      0:00 ps
/ #
```

Let's kill the container now:

```console
$ curl -X POST -d 'cmd=kill' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://shim/
{
  "id": "alpine-runc",
  "runtime": "runc",
  "state": {
    "ociVersion": "1.0.2-dev",
    "id": "alpine-runc",
    "status": "running",
    "pid": 4363,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {}
}
```

The state should be updated after the container process has exited:

```console
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://shim/
{
  "id": "alpine-runc",
  "runtime": "runc",
  "state": {
    "ociVersion": "1.0.2-dev",
    "id": "alpine-runc",
    "status": "stopped",
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {
    "exitStatus": 123,
    "exited": true
  }
}
```

We can now delete the container and terminate the shim:

```console
$ curl -X POST -d 'cmd=delete' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://shim/
$ curl -X DELETE --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://shim/
BYE
```

## Advanced usage

Yacs has many configuration flags (options). This section describes some of them.

### `--container-log-file`

Yacs uses this log file to write the standard output (and error) of the container process. Each line is appended to the log file as a JSON object (also described in a previous section above):

```json
{"m":"[Sun Jun 12 11:51:44 UTC 2022] Hello!","s":"stdout","t":"2022-06-12T11:51:44.947554491Z"}
```

### `--exit-command`

When the container process exits, Yacs will call an "exit command" when `--exit-command` is specified. It is also possible to specify command arguments with `--exit-command-arg`.

This can be useful for daemon-less container managers (e.g., [Yaman][] configures Yacs to call `yaman container cleanup` when a container process exits so that (1) Yaman is notified of this event and (2) it can perform some clean-up tasks).

### `--runtime`

The OCI runtime to use. By default, [Yacr][] will be used.

### `--stdio-dir`

This is the directory where Yacs will create the FIFOs (stdio named pipes).

[jq]: https://stedolan.github.io/jq/
[runc]: https://github.com/opencontainers/runc/
[ttrpc]: https://github.com/containerd/ttrpc
[yacr]: ../yacr/README.md
[yaman]: ../yaman/README.md
