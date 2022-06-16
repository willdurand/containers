# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][].

## Commands

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

⚠️ You must have "root" privileges to use `yaman` because the tool needs to mount the "rootfs" of a container as an Overlay FS. You can use `sudo yaman` as shown in the next sections.

### `yaman image`

Manage OCI images.

#### `yaman image pull`

```
$ yaman image pull docker.io/library/hello-world
downloaded docker.io/library/hello-world:latest
```

#### `yaman image list`

```
$ sudo yaman image list
NAME                  TAG         CREATED                PULLED           REGISTRY
library/hello-world   latest      2021-09-23T23:47:57Z   12 minutes ago   docker.io
library/redis         latest      2022-06-13T20:08:18Z   10 minutes ago   docker.io
```

## Completion

```
$ yaman completion bash | sudo tee /etc/bash_completion.d/yaman > /dev/null
```

[docker]: https://docs.docker.com/reference/
[podman]: https://docs.podman.io/en/latest/
