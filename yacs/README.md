# Yet Another Container Shim

This is an example of a container shim that exposes an HTTP API[^1] to control the lifecycle of a container process. Theoretically, shims should be a small as possible because container managers use a shim per container process. This isn't the case of this shim, though, but also no one should be using it except for learning purposes.

[^1]: this will likely change in the future (to a GRPC API with [ttrpc][])

## Getting started with an example

First, we need a new bundle:

```
$ make alpine_bundle
```

Let's edit the `config.json` file generated in `/tmp/alpine-bundle` as follows:

```diff
--- a/config.json
+++ b/config.json
@@ -1,13 +1,13 @@
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
+                       "sh", "/hello-loop.sh"
                ],
                "env": [
                        "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
@@ -183,4 +183,4 @@
                        "/proc/sysrq-trigger"
                ]
        }
```

We should also create this new file named `hello-loop.sh`:

```
$ cat <<'EOF' > /tmp/alpine-bundle/rootfs/hello-loop.sh
#!/bin/sh

signal_handler() {
    echo "bye, bye"
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

Now we can manually execute `yacs` (the shim) to create a new container with the `yacr` runtime (the default). This step is usually performed by a container manager.

```
$ yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
/home/gitpod/.run/yacs/alpine-1/shim.sock
```

This should start a new shim that will automatically create a container process. We can check with `yacr list` and `ps`:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    created     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle
```

```
$ ps auxf
USER         PID    %CPU %MEM    VSZ     RSS TTY       STAT START   TIME COMMAND
[...]
gitpod       44458  0.0  0.0     1079856 7260 ?        Ssl  22:01   0:00 yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
gitpod       44488  0.0  0.0     1076520 5780 ?        Sl   22:01   0:00  \_ yacr create container --root /home/gitpod/.run/yacr --log-format json --log /home/gitpod/.run/yacs/alpine-1/yacr.log alpine-1
```

When the command returns, it prints a unix socket address that can be used to query the shim using... HTTP. This isn't great but it is enough to demonstrate how a shim works.

We can use `curl` to interact with the shim:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
  "runtime": "yacr",
  "state": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "created",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {}
}
```

We can now start the container by sending the `start` command (`cmd`) in a `POST` HTTP request:

```
$ curl -X POST -d 'cmd=start' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
  "runtime": "yacr",
  "state": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "running",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {}
}
```

> Note: [`jq`][jq] was used to pretty-print the JSON responses in the different examples.

The container is now running, which we can confirm with `yacr list` and `ps`:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    running     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle
```

```
$ ps auxf
USER         PID    %CPU %MEM    VSZ     RSS TTY       STAT START   TIME COMMAND
[...]
gitpod       44458  0.0  0.0     1079856 7260 ?        Ssl  22:01   0:00 yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
gitpod       44488  0.0  0.0     1596    4    ?        S    22:01   0:00  \_ sh /hello-loop.sh
gitpod       55758  0.0  0.0     1596    4    ?        S    22:02   0:00      \_ sleep 1
```

We can query the shim to get the standard output logs:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/stdout
[Fri Jun  3 22:03:01 UTC 2022] Hello!
[Fri Jun  3 22:03:02 UTC 2022] Hello!
[Fri Jun  3 22:03:03 UTC 2022] Hello!
[Fri Jun  3 22:03:04 UTC 2022] Hello!
[Fri Jun  3 22:03:05 UTC 2022] Hello!
```

We can also use the shim HTTP API to send a signal to the container:

```
$ curl -X POST -d 'cmd=kill' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
  "runtime": "yacr",
  "state": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "running",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {}
}
```

Weird, it doesn't look like anything as changed. Let's query the `stdout` logs again:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/stdout
[...]
[Fri Jun  3 22:03:42 UTC 2022] Hello!
[Fri Jun  3 22:03:43 UTC 2022] Hello!
bye, bye
```

The container printed the message of the `signal_handler` defined in the `hello-loop.sh` script so the container should have exited. We can verify by querying the state of the shim again. This time, the container is marked as `stopped` and we have information in the `status` property:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
  "runtime": "yacr",
  "state": {
    "ociVersion": "1.0.2",
    "id": "alpine-1",
    "status": "stopped",
    "pid": 44488,
    "bundle": "/tmp/alpine-bundle"
  },
  "status": {
    "exitStatus": 123,
    "exited": true
  }
}
```

The `exitStatus` is `123` and matches what we defined in the `hello-loop.sh` file created previously. Note also that the shim is still alive and we still have access to the container's full state and stdout/stderr logs. This is one of the reasons why shims are used.

We can now delete the container. This API request should not return anything (HTTP 204):

```
$ curl -X POST -d 'cmd=delete' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
```

If we query the state of the shim again, it should indicate that the container does not exist anymore:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
container 'alpine-1' does not exist
```

Finally, we can terminate the shim with a `DELETE` HTTP request:

```
$ curl -X DELETE --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
BYE
```

## Getting started with `runc`

This shim should be able to use any OCI-compliant runtime like [`runc`][runc] (the reference implementation). Let's reproduce what was done in the previous section but with `runc`.

```
$ yacs --bundle /tmp/alpine-bundle/ --container-id alpine-runc --runtime runc
/home/gitpod/.run/yacs/alpine-runc/shim.sock
```

The `ps` output below shows that `runc` has been invoked:

```
$ ps auxf
USER         PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
[...]
gitpod      4321  0.0  0.0 1079876 6776 ?        Ssl  10:57   0:00 yacs --bundle /tmp/alpine-bundle/ --container-id alpine-runc --runtime runc
gitpod      4363  0.0  0.0 1083784 9980 ?        Ssl  10:57   0:00  \_ runc init
```

```
$ curl -X POST -d 'cmd=start' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://localhost/
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

```
$ runc list
ID            PID         STATUS      BUNDLE               CREATED                          OWNER
alpine-runc   4363        running     /tmp/alpine-bundle   2022-06-05T10:57:18.334165274Z   gitpod
```

Since `runc` is the reference implementation and a production-ready runtime, it has a LOT more features than [`yacr`](../yacr/). For instance, we can use `runc exec` to execute a new process in the container, like spawning a shell:

```
$ $ runc exec -t alpine-runc /bin/sh
/ # ps
PID   USER     TIME  COMMAND
    1 root      0:00 sh /hello-loop.sh
  229 root      0:00 /bin/sh
  236 root      0:00 sleep 1
  237 root      0:00 ps
/ #
```

Let's kill the container now:

```
$ curl -X POST -d 'cmd=kill' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://localhost/
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

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://localhost/
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

```
$ curl -X POST -d 'cmd=delete' --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://localhost/
$ curl -X DELETE --unix-socket /home/gitpod/.run/yacs/alpine-runc/shim.sock http://localhost/
BYE
```

[jq]: https://stedolan.github.io/jq/
[runc]: https://github.com/opencontainers/runc/
[ttrpc]: https://github.com/containerd/ttrpc
