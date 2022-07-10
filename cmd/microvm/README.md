# MicroVM

## Getting started with Docker

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

Let's create a new Docker daemon with the `microvm` runtime registered:

```console
$ ./extras/docker/scripts/run-dockerd
```

In another terminal, you can connect to this daemon by running `docker` with `-H unix:///tmp/d2/d2.socket` or use the `./extras/docker/scripts/docker` wrapper in this repository:

```console
$ ./extras/docker/scripts/docker info
[...]
 Runtimes: io.containerd.runc.v2 io.containerd.runtime.v1.linux microvm runc yacr
 Default Runtime: yacr
[...]
```

We can then use Docker with the `--runtime=microvm` option (because the default
runtime is [Yacr][]):

```console
$ ./extras/docker/scripts/docker run --runtime=microvm --rm hello-world
Unable to find image 'hello-world:latest' locally
latest: Pulling from library/hello-world
2db29710123e: Pull complete
Digest: sha256:13e367d31ae85359f42d637adf6da428f76d75dc9afeb3c21faea0d976f5c651
Status: Downloaded newer image for hello-world:latest

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

## Getting started with Yaman

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

This is an example of what you could do with [Yaman][]:

```console
$ echo 'wttr.in' \
  | sudo yaman c run --rm --interactive docker.io/library/alpine -- xargs wget -qO /dev/stdout \
  | sudo yaman c run --interactive --runtime microvm quay.io/aptible/alpine -- head -n 7
Weather report: Brussels, Belgium

     \  /       Partly cloudy
   _ /"".-.     17 °C
     \_(   ).   ↘ 24 km/h
     /(___(__)  10 km
                0.0 mm

$ sudo yaman c ls -a
CONTAINER ID                       IMAGE                           COMMAND     CREATED          STATUS                      PORTS
26127b728da94fd7a184549f2c0f586c   quay.io/aptible/alpine:latest   head -n 7   15 seconds ago   Exited (0) 12 seconds ago

$ sudo yaman c logs 26127b728da94fd7a184549f2c0f586c
Weather report: Brussels, Belgium

     \  /       Partly cloudy
   _ /"".-.     +22(24) °C
     \_(   ).   ↓ 7 km/h
     /(___(__)  10 km
                0.0 mm
```

[yacr]: ../yacr/README.md
[yaman]: ../yaman/README.md
