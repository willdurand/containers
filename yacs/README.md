# Yet Another Container Shim

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

This should start a new shim that will automatically create a container process with `yacr` and we can check with `yacr list` and `ps`:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    created     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle

$ ps auxf
USER         PID    %CPU %MEM    VSZ     RSS TTY       STAT START   TIME COMMAND
[...]
gitpod       44458  0.0  0.0     1079856 7260 ?        Ssl  22:01   0:00 yacs --bundle=/tmp/alpine-bundle --container-id=alpine-1
gitpod       44488  0.0  0.0     1076520 5780 ?        Sl   22:01   0:00  \_ yacr create container --root /home/gitpod/.run/yacr --log-format text alpine-1
```

When the command returns, it prints a unix socket address that can be used to query the shim using... HTTP. This isn't great but it is enough to demonstrate how a shim works. We can use `curl` to interact with the shim:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
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

> Note: the lifetime of the shim is, at the very least, bound to the lifetime of the container process. When the container process exits, the shim knows it because it waits for the termination of the container process and we'll see more information in the `status` field above.

We can now start the container. We could use `yacr start` but given that we are describing how the `yacs` shim works, let's continue with the HTTP API. We should send the `start` command (`cmd`) using the `POST` HTTP verb:

```
$ curl -X POST -d 'cmd=start' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
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

> Note: the output of the `curl` command is piped to [`jq`][jq].

The container is now running, which we can confirm with `yacr list` and/or `ps`:

```
$ yacr list
ID          STATUS      CREATED                PID         BUNDLE
alpine-1    running     2022-06-03T22:00:00Z   44488       /tmp/alpine-bundle

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

Let's now send a signal to the container:

```
$ curl -X POST -d 'cmd=kill' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
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

It doesn't look like anything as changed so let's query the `stdout` logs again:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/stdout
[...]
[Fri Jun  3 22:03:42 UTC 2022] Hello!
[Fri Jun  3 22:03:43 UTC 2022] Hello!
bye, bye
```

The container has exited according to the logs and we can verify by querying the state of the shim again. This time, the container is marked as `stopped` and we have information in the `status` property:

```
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
{
  "id": "alpine-1",
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

We can now delete the container. This API request should not return anything (HTTP 204). If we query the state of the shim again, it should indicate that the container does not exist anymore:

```
$ curl -X POST -d 'cmd=delete' --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
$ curl --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
container 'alpine-1' not found
```

We can terminate the shim with a `DELETE` HTTP request:

```
$ curl -X DELETE --unix-socket /home/gitpod/.run/yacs/alpine-1/shim.sock http://localhost/
bye
```

[jq]: https://stedolan.github.io/jq/
