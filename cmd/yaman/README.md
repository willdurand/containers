# Yet Another (container) MANager

Yaman is a daemon-less container manager inspired by [Docker][] and [Podman][].

## Commands

**👋 Make sure to [follow these instructions](../../README.md#building-this-project) first.**

⚠️ You must have "root" privileges to use `yaman` because the tool needs to mount the "rootfs" of a container as an Overlay FS. You can use `sudo yaman` as shown in the next sections.

### `yaman image`

Manage OCI images.

#### `yaman image list`

```
$ sudo yaman image list
NAME             TAG         CREATED                PULLED         REGISTRY
library/alpine   latest      2022-05-23T19:19:31Z   34 hours ago   docker.io
library/redis    latest      2022-06-08T18:34:43Z   47 hours ago   docker.io
```

[docker]: https://docs.docker.com/reference/
[podman]: https://docs.podman.io/en/latest/
